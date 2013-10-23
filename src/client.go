package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"

	//"code.google.com/p/go.crypto/ssh/terminal"
)

type (
	Client struct{}
)

const (
	STDIN_FD  = 0
	STDOUT_FD = 1
	STDERR_FD = 2
)

func fail(format string, args ...interface{}) {
	fmt.Printf("\033[%vm%v\033[0m\n", RED, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (this *Client) send(msg Message) error {
	//fmt.Printf("CLIENT DEBUG: msg=%v\n", msg)
	// Open a tunnel if necessary
	/*if terminal.IsTerminal(STDOUT_FD) {
		fmt.Print("HEY DUDE, I CAN TELL THIS IS RUNNING IN A TERMINAL\n")
	} else {
		fmt.Print("HEY DUDE, I COULD TELL DIZ AIN'T NO TERMNAL\n")
	}*/

	if !strings.Contains(strings.ToLower(sshHost), "localhost") && !strings.Contains(strings.ToLower(sshHost), "127.0.0.1") {
		bs, err := exec.Command("hostname").Output()
		if err != nil || !bytes.HasPrefix(bs, []byte("ip-")) {
			t, err := OpenTunnel()
			if err != nil {
				return err
			}
			defer t.Close()
		}
	}

	conn, err := net.Dial("tcp", "localhost:9999")
	if err != nil {
		return err
	}
	defer conn.Close()

	err = Send(conn, msg)
	if err != nil {
		return err
	}

	for {
		msg, err := Receive(conn)
		if err != nil {
			break
		}
		switch msg.Type {
		case ReadLineRequest:
			fmt.Printf("\033[%vm%v\033[0m", RED, msg.Body)
			response, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				fmt.Printf("local error, operation aborted: \033[%vm%v\033[0m\n", RED, err)
				os.Exit(1)
			}
			Send(conn, Message{ReadLineResponse, response})
		case Hijack:
			ec := make(chan error, 1)
			go func() {
				_, err := io.Copy(conn, os.Stdin)
				ec <- err
			}()
			go func() {
				_, err := io.Copy(os.Stdout, conn)
				ec <- err
			}()
			return <-ec
		case Log:
			fmt.Printf("%v", msg.Body)
		case Error:
			fmt.Printf("\033[%vm%v\033[0m\n", RED, msg.Body)
			os.Exit(1)
		default:
			log.Printf("received %v", msg)
		}
	}

	return nil
}

func (this *Client) Do(args []string) {
	local := &Local{}
	localType := reflect.TypeOf(local)

	for _, cmd := range commands {
		if args[1] == cmd.ShortName || args[1] == cmd.LongName {
			parsed, err := cmd.Parse(args[2:])
			if err != nil {
				fail("%v", err)
				return
			}

			if method, ok := localType.MethodByName(cmd.ServerName); ok {
				vs := []reflect.Value{reflect.ValueOf(local)}
				for _, v := range parsed {
					vs = append(vs, reflect.ValueOf(v))
				}
				vs = method.Func.Call(vs)

				// Handle an error being returned
				if len(vs) > 0 && vs[0].CanInterface() {
					err, ok = vs[0].Interface().(error)
					if ok {
						fail("%v", err)
						return
					}
				}

				return
			}

			/*if cmd.LongName == "logger" {
				err = (&Local{}).Logger(args[0], args[1], parsed[2])
				if err != nil {
					fail("%v", err)
				}
				return
			}*/

			bs, _ := json.Marshal(append([]interface{}{cmd.ServerName}, parsed...))
			err = this.send(Message{Call, string(bs)})
			if err != nil {
				fail("%v", err)
				return
			}
			return
		}
	}

	fail("Unknown command `%v`", args[1])
}
