package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"

	"github.com/Sendhub/logserver/server"
)

type (
	Server struct {
		LogServer                 *server.Server
		currentLoadBalancerConfig string
	}
)

var (
	globalLock sync.Mutex
	appLocks   = map[string]bool{}
)

func run(name string, args ...string) error {
	log.Printf("= %v %v", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (this *Server) getLogger(conn net.Conn) io.Writer {
	return NewTimeLogger(NewMessageLogger(conn))
}

func (this *Server) getTitleAndDimLoggers(conn net.Conn) (io.Writer, io.Writer) {
	logger := this.getLogger(conn)
	titleLogger := NewFormatter(logger, GREEN)
	dimLogger := NewFormatter(logger, DIM)
	return titleLogger, dimLogger
}

// Provides common functionality for appending tokenized unique items to a list, with logging of details regarding which items were added
// or rejected.  Helps avoid repetition in some of the cmd_* methods.
func (this *Server) UniqueStringsAppender(conn net.Conn, items []string, addItems []string, itemType string, addListenerFn func(string)) []string {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Adding %vs\n", itemType)

	for _, addItem := range addItems {
		if len(addItem) == 0 {
			continue
		}
		found := false
		for _, existingItem := range items {
			if strings.ToLower(addItem) == strings.ToLower(existingItem) {
				fmt.Fprintf(dimLogger, "%v already exists: %v\n", strings.Title(itemType), addItem)
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(dimLogger, "Adding %v: %v\n", itemType, addItem)
			items = append(items, addItem)
			if addListenerFn != nil {
				addListenerFn(addItem)
			}
		}
	}
	return items
}

// Provides common functionality for removing tokenized items from a list, with logging of details regarding which items were removed.
// Helps avoid repetition in some of the cmd_* methods.
func (this *Server) UniqueStringsRemover(conn net.Conn, items []string, removeItems []string, itemType string, removeListenerFn func(string)) []string {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Removing %vs\n", itemType)

	originalItems := items
	items = []string{}
	for _, existingItem := range originalItems {
		keep := true
		for _, removeItem := range removeItems {
			if strings.ToLower(removeItem) == strings.ToLower(existingItem) {
				fmt.Fprintf(dimLogger, "Removing %v: %v\n", itemType, existingItem)
				keep = false
				break
			}
		}
		if keep {
			items = append(items, existingItem)
		} else if removeListenerFn != nil {
			removeListenerFn(existingItem)
		}
	}
	return items
}

func (this *Server) handleCall(conn net.Conn, body string) error {
	var args []interface{}
	err := json.Unmarshal([]byte(body), &args)
	if err != nil {
		return err
	}

	// Convert any args of type T=map[string]interface{} to map[string]string.
	for i, arg := range args {
		m, ok := arg.(map[string]interface{})
		if ok {
			nMap := map[string]string{}
			for k, v := range m {
				nMap[k] = fmt.Sprint(v)
			}
			args[i] = nMap
		}
	}
	// Convert multiple string args to []string.
	for i, arg := range args {
		list, ok := arg.([]interface{})
		if ok {
			nList := []string{}
			for _, value := range list {
				nList = append(nList, fmt.Sprint(value))
			}
			args[i] = nList
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("expected command")
	}
	fmt.Printf("received cmd: %v\n", args)
	for _, cmd := range commands {
		if cmd.ServerName == args[0].(string) {
			method, ok := reflect.TypeOf(this).MethodByName(args[0].(string))
			if !ok {
				return fmt.Errorf("unknown method: %v", cmd)
			}
			values := make([]reflect.Value, len(args)+1)
			values[0] = reflect.ValueOf(this)
			values[1] = reflect.ValueOf(conn)
			for i := 1; i < len(args); i++ {
				values[i+1] = reflect.ValueOf(args[i])
			}
			defer func() {
				// reflect can panic so recover here
				if r := recover(); r != nil {
					Errorf(conn, "error running command: %v, %v", args, r)
				}
			}()
			// For any application specific write commands we lock
			//   based on the application name
			needsLock := cmd.AppWrite
			if !needsLock {
				// also lock these, lock is based on git directory
				needsLock = cmd.LongName == "pre-receive" || cmd.LongName == "post-receive"
			}
			if needsLock && args[1] != "" {
				globalLock.Lock()
				active, ok := appLocks[args[1].(string)]
				if ok && active {
					globalLock.Unlock()
					return fmt.Errorf("a command is already running for this application")
				}
				appLocks[args[1].(string)] = true
				globalLock.Unlock()
				// Remove lock when we're done
				defer func() {
					globalLock.Lock()
					delete(appLocks, args[1].(string))
					globalLock.Unlock()
				}()
			}
			values = method.Func.Call(values)

			// Handle an error being returned
			if len(values) >= 0 && values[0].CanInterface() {
				err, ok = values[0].Interface().(error)
				if ok {
					return err
				}
			}

			return nil
		}
	}
	return fmt.Errorf("unknown command: %v", args[1])
}

func (this *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := Receive(conn)
	if err != nil {
		log.Printf("invalid message: %v", err)
		Send(conn, Message{Error, "Error reading message"})
		return
	}
	log.Printf("received: %v", msg)
	switch msg.Type {
	case Call:
		err = this.handleCall(conn, msg.Body)
		if err != nil {
			Send(conn, Message{Error, err.Error()})
		}
	}
}

func (this *Server) verifyRequiredBuildPacks() error {
	return this.WithConfig(func(cfg *Config) error {
		for _, app := range cfg.Applications {
			_, ok := BUILD_PACKS[app.BuildPack]
			if !ok {
				return fmt.Errorf("fatal: missing build-pack '%v' for application '%v'", app.BuildPack, app.Name)
			}
		}
		return nil
	})
}

func (this *Server) start() error {
	var err error

	this.LogServer, err = server.Start()
	if err != nil {
		return err
	}

	initDrains(this)
	go this.monitorFreeMemory()

	log.Println("starting server on :9999")
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		return err
	}

	err = this.verifyRequiredBuildPacks()
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("err in connection loop: %v", err)
			continue
		}
		log.Printf("new connection %v", conn.RemoteAddr())
		go this.handleConnection(conn)
	}
	return nil
}
