package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_TIMEOUT = 30000 // default timeout for operations (in milliseconds)
)

var (
	user                string
	haveKeyring         bool
	keyring             ssh.ClientAuth
	connectedHosts      map[string]*ssh.ClientConn
	connectedHostsMutex sync.Mutex
	repliesChan         chan interface{}
	requestsChan        chan *ProxyRequest
)

type (
	MegaPassword struct {
		pass string
	}

	SignerContainer struct {
		signers []ssh.Signer
	}

	SshResult struct {
		hostname string
		stdout   string
		stderr   string
		err      error
	}

	ScpResult struct {
		hostname string
		err      error
	}

	ProxyRequest struct {
		Action   string
		Password string // password for private key (only for Action == "password")
		Cmd      string // command to execute (only for Action == "ssh")
		Source   string // source file to copy (only for Action == "scp")
		Target   string // target file (only for Action == "scp")
		Hosts    []string
		Timeout  uint64
	}

	Reply struct {
		Hostname string
		Stdout   string
		Stderr   string
		Success  bool
		ErrMsg   string
	}

	PasswordRequest struct {
		PasswordFor string
	}

	FinalReply struct {
		TotalTime     float64
		TimedOutHosts map[string]bool
	}

	ConnectionProgress struct {
		ConnectedHost string
	}

	UserError struct {
		IsCritical bool
		ErrorMsg   string
	}

	InitializeComplete struct {
		InitializeComplete bool
	}

	DisableReportConnectedHosts bool
	EnableReportConnectedHosts  bool
)

func (t *SignerContainer) Key(i int) (key ssh.PublicKey, err error) {
	if i >= len(t.signers) {
		return
	}

	key = t.signers[i].PublicKey()
	return
}

func (t *SignerContainer) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	if i >= len(t.signers) {
		return
	}

	sig, err = t.signers[i].Sign(rand, data)
	return
}

func (t *MegaPassword) Password(user string) (password string, err error) {
	fmt.Println("User ", user)
	password = t.pass
	return
}

func reportErrorToUser(msg string) {
	repliesChan <- &UserError{ErrorMsg: msg}
}

func reportCriticalErrorToUser(msg string) {
	repliesChan <- &UserError{IsCritical: true, ErrorMsg: msg}
}

func makeConfig() *ssh.ClientConfig {
	clientAuth := []ssh.ClientAuth{}

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock != "" {
		for {
			sock, err := net.Dial("unix", sshAuthSock)
			if err != nil {
				netErr := err.(net.Error)
				if netErr.Temporary() {
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
					continue
				}

				reportErrorToUser("Cannot open connection to SSH agent: " + netErr.Error())
			} else {
				agent := ssh.NewAgentClient(sock)
				identities, err := agent.RequestIdentities()
				if err != nil {
					reportErrorToUser("Cannot request identities from ssh-agent: " + err.Error())
				} else if len(identities) > 0 {
					clientAuth = append(clientAuth, ssh.ClientAuthAgent(agent))
				}
			}

			break
		}
	}

	if haveKeyring {
		clientAuth = append(clientAuth, keyring)
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: clientAuth,
	}
}

func makeSigner(keyname string) (signer ssh.Signer, err error) {
	fp, err := os.Open(keyname)
	if err != nil {
		if !os.IsNotExist(err) {
			reportErrorToUser("Could not parse " + keyname + ": " + err.Error())
		}
		return
	}
	defer fp.Close()

	buf, err := ioutil.ReadAll(fp)
	if err != nil {
		reportErrorToUser("Could not read " + keyname + ": " + err.Error())
		return
	}

	if bytes.Contains(buf, []byte("ENCRYPTED")) {
		var (
			tmpfp *os.File
			out   []byte
		)

		tmpfp, err = ioutil.TempFile("", "key")
		if err != nil {
			reportErrorToUser("Could not create temporary file: " + err.Error())
			return
		}

		tmpName := tmpfp.Name()

		defer func() { tmpfp.Close(); os.Remove(tmpName) }()

		_, err = tmpfp.Write(buf)

		if err != nil {
			reportErrorToUser("Could not write encrypted key contents to temporary file: " + err.Error())
			return
		}

		err = tmpfp.Close()
		if err != nil {
			reportErrorToUser("Could not close temporary file: " + err.Error())
			return
		}

		repliesChan <- &PasswordRequest{PasswordFor: keyname}
		response := <-requestsChan

		if response.Password == "" {
			reportErrorToUser("No passphase supplied in request for " + keyname)
			err = errors.New("No passphare supplied")
			return
		}

		cmd := exec.Command("ssh-keygen", "-f", tmpName, "-N", "", "-P", response.Password, "-p")
		out, err = cmd.CombinedOutput()
		if err != nil {
			reportErrorToUser(strings.TrimSpace(string(out)))
			return
		}

		tmpfp, err = os.Open(tmpName)
		if err != nil {
			reportErrorToUser("Cannot open back " + tmpName)
			return
		}

		buf, err = ioutil.ReadAll(tmpfp)
		if err != nil {
			return
		}

		tmpfp.Close()
		os.Remove(tmpName)
	}

	signer, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		reportErrorToUser("Could not parse " + keyname + ": " + err.Error())
		return
	}

	return
}

func makeKeyring() {
	signers := []ssh.Signer{}
	keys := []string{os.Getenv("HOME") + "/.ssh/id_rsa", os.Getenv("HOME") + "/.ssh/id_dsa"}

	for _, keyname := range keys {
		signer, err := makeSigner(keyname)
		if err == nil {
			signers = append(signers, signer)
		}
	}

	if len(signers) == 0 {
		haveKeyring = false
	} else {
		haveKeyring = true
		keyring = ssh.ClientAuthKeyring(&SignerContainer{signers})
	}
}

func getConnection(hostname string) (conn *ssh.ClientConn, err error) {
	connectedHostsMutex.Lock()
	conn = connectedHosts[hostname]
	connectedHostsMutex.Unlock()
	if conn != nil {
		return
	}

	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New("Panic: " + fmt.Sprint(msg))
		}
	}()
	conn, err = ssh.Dial("tcp", hostname+":22", makeConfig())
	if err != nil {
		return
	}

	sendProxyReply(&ConnectionProgress{ConnectedHost: hostname})

	connectedHostsMutex.Lock()
	connectedHosts[hostname] = conn
	connectedHostsMutex.Unlock()

	return
}

func uploadFile(target string, contents []byte, hostname string) (stdout, stderr string, err error) {
	conn, err := getConnection(hostname)
	if err != nil {
		return
	}

	session, err := conn.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	cmd := "cat >" + target
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Start(cmd)
	if err != nil {
		return
	}

	_, err = stdinPipe.Write(contents)
	if err != nil {
		return
	}

	err = stdinPipe.Close()
	if err != nil {
		return
	}

	err = session.Wait()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	return
}

func executeCmd(cmd string, hostname string) (stdout, stderr string, err error) {
	conn, err := getConnection(hostname)
	if err != nil {
		return
	}

	session, err := conn.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err = session.Run(cmd)

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	return
}

func initialize() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	user = os.Getenv("LOGNAME")

	repliesChan = make(chan interface{})
	requestsChan = make(chan *ProxyRequest)

	go inputDecoder()
	go jsonReplierThread()

	makeKeyring()
	connectedHosts = make(map[string]*ssh.ClientConn)
}

func jsonReplierThread() {
	connectionReporting := true

	for {
		reply := <-repliesChan

		switch reply.(type) {
		case DisableReportConnectedHosts:
			connectionReporting = false

			continue

		case EnableReportConnectedHosts:
			connectionReporting = true
			continue

		case *ConnectionProgress:
			if !connectionReporting {
				continue
			}
		}

		buf, err := json.Marshal(reply)
		if err != nil {
			panic("Could not marshal json reply: " + err.Error())
		}

		fmt.Println(string(buf))
	}
}

func sendProxyReply(response interface{}) {
	repliesChan <- response
}

func debug(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func runAction(msg *ProxyRequest) {
	var executeFunc func(string) *SshResult

	if msg.Action == "ssh" {
		if msg.Cmd == "" {
			reportCriticalErrorToUser("Empty 'Cmd'")
			return
		}

		executeFunc = func(hostname string) *SshResult {
			stdout, stderr, err := executeCmd(msg.Cmd, hostname)
			return &SshResult{hostname: hostname, stdout: stdout, stderr: stderr, err: err}
		}
	} else if msg.Action == "scp" {
		if msg.Source == "" {
			reportCriticalErrorToUser("Empty 'Source'")
			return
		}

		if msg.Target == "" {
			reportCriticalErrorToUser("Empty 'Target'")
			return
		}

		fp, err := os.Open(msg.Source)
		if err != nil {
			reportCriticalErrorToUser(err.Error())
			return
		}

		defer fp.Close()

		contents, err := ioutil.ReadAll(fp)
		if err != nil {
			reportCriticalErrorToUser("Cannot read " + msg.Source + " contents: " + err.Error())
			return
		}

		executeFunc = func(hostname string) *SshResult {
			stdout, stderr, err := uploadFile(msg.Target, contents, hostname)
			return &SshResult{hostname: hostname, stdout: stdout, stderr: stderr, err: err}
		}
	}

	timeout := uint64(DEFAULT_TIMEOUT)

	if msg.Timeout > 0 {
		timeout = msg.Timeout
	}

	startTime := time.Now().UnixNano()

	responseChannel := make(chan *SshResult, 10)
	timeoutChannel := time.After(time.Millisecond * time.Duration(timeout))

	timedOutHosts := make(map[string]bool)

	sendProxyReply(EnableReportConnectedHosts(true))

	for _, hostname := range msg.Hosts {
		timedOutHosts[hostname] = true

		go func(host string) {
			responseChannel <- executeFunc(host)
		}(hostname)
	}

	for i := 0; i < len(msg.Hosts); i++ {
		select {
		case <-timeoutChannel:
			goto finish
		case msg := <-responseChannel:
			delete(timedOutHosts, msg.hostname)
			success := true
			errMsg := ""
			if msg.err != nil {
				errMsg = msg.err.Error()
				success = false
			}
			sendProxyReply(Reply{Hostname: msg.hostname, Stdout: msg.stdout, Stderr: msg.stderr, ErrMsg: errMsg, Success: success})
		}
	}

finish:

	connectedHostsMutex.Lock()
	for hostname, _ := range timedOutHosts {
		if conn, ok := connectedHosts[hostname]; ok {
			conn.Close()
		}

		delete(connectedHosts, hostname)
	}
	connectedHostsMutex.Unlock()

	sendProxyReply(DisableReportConnectedHosts(true))

	sendProxyReply(FinalReply{TotalTime: float64(time.Now().UnixNano()-startTime) / 1e9, TimedOutHosts: timedOutHosts})
}

func inputDecoder() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := new(ProxyRequest)

		line := scanner.Bytes()
		err := json.Unmarshal(line, msg)
		if err != nil {
			reportCriticalErrorToUser("Cannot parse JSON: " + err.Error())
			continue
		}

		requestsChan <- msg
	}

	if err := scanner.Err(); err != nil {
		reportCriticalErrorToUser("Error reading stdin: " + err.Error())
	}

	close(requestsChan)
}

func runProxy() {
	for msg := range requestsChan {
		switch {
		case msg.Action == "ssh" || msg.Action == "scp":
			runAction(msg)
		default:
			reportCriticalErrorToUser("Unsupported action: " + msg.Action)
		}
	}
}

func main() {
	initialize()
	sendProxyReply(&InitializeComplete{InitializeComplete: true})
	runProxy()
}
