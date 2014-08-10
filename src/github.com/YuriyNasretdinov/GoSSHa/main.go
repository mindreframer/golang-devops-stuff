package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/gosshold/ssh"
	"encoding/json"
	"errors"
	"flag"
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
	DEFAULT_TIMEOUT               = 30000 // default timeout for operations (in milliseconds)
	CHUNK_SIZE                    = 65536 // chunk size in bytes for scp
	THROUGHPUT_SLEEP_INTERVAL     = 100   // how many milliseconds to sleep between writing "tickets" to channel in maxThroughputThread
	MIN_CHUNKS                    = 10    // minimum allowed count of chunks to be sent per sleep interval
	MIN_THROUGHPUT                = CHUNK_SIZE * MIN_CHUNKS * (1000 / THROUGHPUT_SLEEP_INTERVAL)
	MAX_OPENSSH_AGENT_CONNECTIONS = 128 // default connection backlog for openssh
)

var (
	user                string
	signers             []ssh.Signer
	keys                []string
	connectedHosts      map[string]*ssh.ClientConn
	connectedHostsMutex sync.Mutex
	repliesChan         = make(chan interface{})
	requestsChan        = make(chan *ProxyRequest)
	maxThroughputChan   = make(chan bool, MIN_CHUNKS) // channel that is used for throughput limiting in scp
	maxThroughput       uint64                        // max throughput (for scp) in bytes per second
	maxThroughputMutex  sync.Mutex
	agentConnChan       = make(chan chan bool) // channel for getting "ticket" for new agent connection
	agentConnFreeChan   = make(chan bool, 10)  // channel for freeing connections
	sshAuthSock         string
)

type (
	SignerContainer struct {
		signers []ssh.Signer
		agentKr ssh.ClientKeyring
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
		Action        string
		Password      string // password for private key (only for Action == "password")
		Cmd           string // command to execute (only for Action == "ssh")
		Source        string // source file to copy (only for Action == "scp")
		Target        string // target file (only for Action == "scp")
		Hosts         []string
		Timeout       uint64 // timeout (in milliseconds), default is DEFAULT_TIMEOUT
		MaxThroughput uint64 // max throughput (for scp) in bytes per second, default is no limit
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
	if i < len(t.signers) {
		key = t.signers[i].PublicKey()
	} else if t.agentKr != nil {
		key, err = t.agentKr.Key(i - len(t.signers))
	}

	return
}

func (t *SignerContainer) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	if i < len(t.signers) {
		sig, err = t.signers[i].Sign(rand, data)
	} else if t.agentKr != nil {
		sig, err = t.agentKr.Sign(i-len(t.signers), rand, data)
	}

	return
}

func reportErrorToUser(msg string) {
	repliesChan <- &UserError{ErrorMsg: msg}
}

func reportCriticalErrorToUser(msg string) {
	repliesChan <- &UserError{IsCritical: true, ErrorMsg: msg}
}

func waitAgent() {
	if sshAuthSock != "" {
		respChan := make(chan bool)
		agentConnChan <- respChan
		<-respChan
	}
}

func releaseAgent() {
	if sshAuthSock != "" {
		agentConnFreeChan <- true
	}
}

func makeConfig() (config *ssh.ClientConfig, agentUnixSock net.Conn) {
	clientAuth := []ssh.ClientAuth{}

	var (
		agentKr ssh.ClientKeyring
		ok      bool
		err     error
	)

	if sshAuthSock != "" {
		for {
			agentUnixSock, err = net.Dial("unix", sshAuthSock)

			if err != nil {
				netErr := err.(net.Error)
				if netErr.Temporary() {
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
					continue
				}

				reportErrorToUser("Cannot open connection to SSH agent: " + netErr.Error())
			} else {
				authAgent := ssh.ClientAuthAgent(ssh.NewAgentClient(agentUnixSock))
				agentKr, ok = authAgent.(ssh.ClientKeyring)
				if !ok {
					reportErrorToUser("Type assertion failed: ssh.ClientAuthAgent no longer returns ssh.ClientKeyring, using fallback")
					clientAuth = append(clientAuth, authAgent)
				}
			}

			break
		}
	}

	keyring := ssh.ClientAuthKeyring(&SignerContainer{signers, agentKr})
	clientAuth = append(clientAuth, keyring)

	config = &ssh.ClientConfig{
		User: user,
		Auth: clientAuth,
	}

	return
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
			reportErrorToUser("No passphrase supplied in request for " + keyname)
			err = errors.New("No passphrase supplied")
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

func makeSigners() {
	signers = []ssh.Signer{}

	for _, keyname := range keys {
		signer, err := makeSigner(keyname)
		if err == nil {
			signers = append(signers, signer)
		}
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

	waitAgent()
	conf, agentConn := makeConfig()
	if agentConn != nil {
		defer agentConn.Close()
	}

	defer releaseAgent()

	port := "22"
	str := strings.SplitN(hostname, ":", 2)
	if len(str) == 2 {
		hostname = str[0]
		port = str[1]
	}

	conn, err = ssh.Dial("tcp", hostname+":"+port, conf)
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

	cmd := "cat >'" + strings.Replace(target, "'", "'\\''", -1) + "'"
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

	for start, maxEnd := 0, len(contents); start < maxEnd; start += CHUNK_SIZE {
		<-maxThroughputChan

		end := start + CHUNK_SIZE
		if end > maxEnd {
			end = maxEnd
		}
		_, err = stdinPipe.Write(contents[start:end])
		if err != nil {
			return
		}
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

// do not allow more than maxConn simultaneous ssh-agent connections
func agentConnectionManagerThread(maxConn int) {
	freeConn := maxConn // free connections count

	for {
		reqCh := agentConnChan
		freeCh := agentConnFreeChan

		if freeConn <= 0 {
			reqCh = nil
		}

		// fmt.Fprintln(os.Stderr, "Free connections: ", freeConn)

		select {
		case respChan := <-reqCh:
			freeConn--
			respChan <- true
		case <-freeCh:
			freeConn++
		}
	}
}

func initialize() {
	var (
		pubKey              string
		maxAgentConnections int
	)

	flag.StringVar(&pubKey, "i", "", "Optional path to public key to use")
	flag.StringVar(&user, "l", os.Getenv("LOGNAME"), "Optional login name")
	flag.IntVar(&maxAgentConnections, "c", MAX_OPENSSH_AGENT_CONNECTIONS, "Maximum simultaneous ssh-agent connections")
	flag.Parse()

	keys = []string{os.Getenv("HOME") + "/.ssh/id_rsa", os.Getenv("HOME") + "/.ssh/id_dsa", os.Getenv("HOME") + "/.ssh/id_ecdsa"}

	if pubKey != "" {
		if strings.HasSuffix(pubKey, ".pub") {
			pubKey = strings.TrimSuffix(pubKey, ".pub")
		}

		keys = append(keys, pubKey)
	}

	sshAuthSock = os.Getenv("SSH_AUTH_SOCK")

	runtime.GOMAXPROCS(runtime.NumCPU())

	if sshAuthSock != "" {
		go agentConnectionManagerThread(maxAgentConnections)
	}

	go inputDecoder()
	go jsonReplierThread()
	go maxThroughputThread()

	makeSigners()
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

		if buf[0] == '{' {
			typeStr := strings.TrimPrefix(fmt.Sprintf("%T", reply), "*main.")
			fmt.Printf("{\"Type\":\"%s\",%s}\n", typeStr, buf[1:len(buf)-1])
		} else {
			fmt.Println(string(buf))
		}
	}
}

func sendProxyReply(response interface{}) {
	repliesChan <- response
}

func debug(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func maxThroughputThread() {
	for {
		maxThroughputMutex.Lock()
		throughput := maxThroughput
		maxThroughputMutex.Unlock()

		// how many chunks can be sent in specified time interval
		chunks := throughput / CHUNK_SIZE * THROUGHPUT_SLEEP_INTERVAL / 1000

		if chunks < MIN_CHUNKS {
			chunks = MIN_CHUNKS
		}

		for i := uint64(0); i < chunks; i++ {
			maxThroughputChan <- true
		}

		if throughput > 0 {
			time.Sleep(THROUGHPUT_SLEEP_INTERVAL * time.Millisecond)
		}
	}
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

		if msg.MaxThroughput > 0 && msg.MaxThroughput < MIN_THROUGHPUT {
			reportErrorToUser(fmt.Sprint("Minimal supported throughput is ", MIN_THROUGHPUT, " Bps"))
		}

		maxThroughputMutex.Lock()
		maxThroughput = msg.MaxThroughput
		maxThroughputMutex.Unlock()

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
			sendProxyReply(&Reply{Hostname: msg.hostname, Stdout: msg.stdout, Stderr: msg.stderr, ErrMsg: errMsg, Success: success})
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

	sendProxyReply(&FinalReply{TotalTime: float64(time.Now().UnixNano()-startTime) / 1e9, TimedOutHosts: timedOutHosts})
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
