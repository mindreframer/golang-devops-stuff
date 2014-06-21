/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Author: Brad Fitzpatrick <brad@danga.com>

// runsit runs stuff.
package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bradfitz/runsit/jsonconfig"
)

// Flags.
var (
	httpPort  = flag.Int("http_port", 4762, "HTTP localhost admin port.")
	configDir = flag.String("config_dir", "/etc/runsit", "Directory containing per-task *.json config files.")
)

var (
	logBuf = new(logBuffer)
	logger = log.New(io.MultiWriter(os.Stderr, logBuf), "", log.Lmicroseconds|log.Lshortfile)
)

const systemLogSize = 64 << 10

// logBuffer is a ring buffer.
type logBuffer struct {
	mu   sync.Mutex
	i    int
	full bool
	buf  [systemLogSize]byte
}

func (b *logBuffer) Write(p []byte) (ntot int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for len(p) > 0 {
		n := copy(b.buf[b.i:], p)
		p = p[n:]
		ntot += n
		b.i += n
		if b.i == len(b.buf) {
			b.i = 0
			b.full = true
		}
	}
	return
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.full {
		return string(b.buf[:b.i])
	}
	s := string(b.buf[b.i:]) + string(b.buf[:b.i])
	if nl := strings.Index(s, "\n"); nl != -1 {
		// Remove first line, since it's probably truncated
		s = s[nl+1:]
	}
	return "...\n" + s
}

// A Task is a named daemon. A single instance of Task exists for the
// life of the runsit daemon, despite how many times the task has
// failed and restarted. (the exception is if the config file for the
// task is deleted, and then the *Task is removed from the global tasks
// map and a new one could appear later with the same name)
type Task struct {
	// Immutable:
	Name     string
	tf       TaskFile
	controlc chan interface{}

	// State owned by loop's goroutine:
	config    jsonconfig.Obj // last valid config
	configErr error          // configuration error
	errTime   time.Time      // of last configErr
	running   *TaskInstance
	failures  []*TaskInstance // last few failures, oldest first.
}

// TaskInstance is a particular instance of a running (or now dead) Task.
type TaskInstance struct {
	task      *Task          // set once; not goroutine safe (may only call public methods)
	startTime time.Time      // set once; immutable
	config    jsonconfig.Obj // set once; immutable
	lr        *LaunchRequest // set once; immutable (actual command parameters)
	cmd       *exec.Cmd      // set once; immutable (command parameters to helper process)
	output    TaskOutput     // internal locking, safe for concurrent access

	// Set (in awaitDeath) when task finishes running:
	endTime time.Time
	waitErr error // typically nil or *exec.ExitError
}

// ID returns a unique ID string for this task instance.
func (in *TaskInstance) ID() string {
	return fmt.Sprintf("%q/%d-pid%d", in.task.Name, in.startTime.Unix(), in.Pid())
}

func (in *TaskInstance) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(fmt.Sprintf("Task %s: %s", in.ID(), format), args...)
	in.output.Add(&Line{
		T:        time.Now(),
		Name:     "system",
		Data:     msg,
		instance: in,
	})
	logger.Print(msg)
}

func (in *TaskInstance) Pid() int {
	if in.cmd == nil || in.cmd.Process == nil {
		return 0
	}
	return in.cmd.Process.Pid
}

func (in *TaskInstance) Output() []*Line {
	return in.output.lineSlice()
}

// TaskOutput is the output of a TaskInstance.
// Only the last maxKeepLines lines are kept.
type TaskOutput struct {
	mu    sync.Mutex
	lines list.List // of *Line
}

func (to *TaskOutput) Add(l *Line) {
	to.mu.Lock()
	defer to.mu.Unlock()
	to.lines.PushBack(l)
	const maxKeepLines = 5000
	if to.lines.Len() > maxKeepLines {
		to.lines.Remove(to.lines.Front())
	}
}

func (to *TaskOutput) lineSlice() []*Line {
	to.mu.Lock()
	defer to.mu.Unlock()
	var lines []*Line
	for e := to.lines.Front(); e != nil; e = e.Next() {
		lines = append(lines, e.Value.(*Line))
	}
	return lines
}

func NewTask(name string) *Task {
	t := &Task{
		Name:     name,
		controlc: make(chan interface{}),
	}
	go t.loop()
	return t
}

func (t *Task) Printf(format string, args ...interface{}) {
	logger.Printf(fmt.Sprintf("Task %q: %s", t.Name, format), args...)
}

func (t *Task) loop() {
	t.Printf("Starting")
	defer t.Printf("Loop exiting")
	for cm := range t.controlc {
		switch m := cm.(type) {
		case statusRequestMessage:
			m.resCh <- t.status()
		case updateMessage:
			t.update(m.tf)
		case stopMessage:
			err := t.stop()
			m.resc <- err
		case instanceGoneMessage:
			t.onTaskFinished(m)
		case restartIfStoppedMessage:
			t.restartIfStopped()
		}
	}
}

// Line is a line of output from a TaskInstance.
type Line struct {
	T    time.Time
	Name string // "stdout", "stderr", or "system"
	Data string // line or prefix of line

	isPrefix bool // truncated line? (too long)
	instance *TaskInstance
}

type updateMessage struct {
	tf TaskFile
}

type stopMessage struct {
	resc chan error
}

type restartIfStoppedMessage struct{}

// instanceGoneMessage is sent when a task instance's process finishes,
// successfully or otherwise. Any error is in instance.waitErr.
type instanceGoneMessage struct {
	in *TaskInstance
}

// statusRequestMessage is sent from the web UI (via the
// RunningInstance accessor) to obtain the task's current status
type statusRequestMessage struct {
	resCh chan<- *TaskStatus
}

func (t *Task) Update(tf TaskFile) {
	t.controlc <- updateMessage{tf}
}

// run in Task.loop
func (t *Task) onTaskFinished(m instanceGoneMessage) {
	m.in.Printf("Task exited; err=%v", m.in.waitErr)
	if m.in == t.running {
		t.running = nil
	}
	const keepFailures = 5
	if len(t.failures) == keepFailures {
		copy(t.failures, t.failures[1:])
		t.failures = t.failures[:keepFailures-1]
	}
	t.failures = append(t.failures, m.in)

	aliveTime := m.in.endTime.Sub(m.in.startTime)
	restartIn := 0 * time.Second
	if min := 5 * time.Second; aliveTime < min {
		restartIn = min - aliveTime
	}

	if m.in.waitErr == nil {
		// TODO: vary restartIn based on whether this instance
		// and the previous few completed successfully or not?
	}

	time.AfterFunc(restartIn, func() {
		t.controlc <- restartIfStoppedMessage{}
	})
}

// run in Task.loop
func (t *Task) restartIfStopped() {
	if t.running != nil || t.config == nil {
		return
	}
	t.Printf("Restarting")
	t.updateFromConfig(t.config)
}

// run in Task.loop
func (t *Task) update(tf TaskFile) {
	t.config = nil
	t.stop()

	fileName := tf.ConfigFileName()
	if fileName == "" {
		t.Printf("config file deleted; stopping")
		DeleteTask(t.Name)
		return
	}

	jc, err := jsonconfig.ReadFile(fileName)
	if err != nil {
		t.configError("Bad config file: %v", err)
		return
	}
	t.updateFromConfig(jc)
}

// run in Task.loop
func (t *Task) configError(format string, args ...interface{}) error {
	t.configErr = fmt.Errorf(format, args...)
	t.errTime = time.Now()
	t.Printf("%v", t.configErr)
	return t.configErr
}

// run in Task.loop
func (t *Task) startError(format string, args ...interface{}) error {
	// TODO: make start error and config error different?
	return t.configError(format, args...)
}

// run in Task.loop
func (t *Task) updateFromConfig(jc jsonconfig.Obj) (err error) {
	t.config = nil
	t.stop()

	env := []string{}
	stdEnv := jc.OptionalBool("standardEnv", true)

	userStr := jc.OptionalString("user", "")
	groupStr := jc.OptionalString("group", "")

	// TODO: medium-term hack to run on linux/arm which lacks cgo support,
	// so let users define these, even though user.Lookup will fail.
	userErrUid := jc.OptionalString("userLookupErrUid", "")
	userErrGid := jc.OptionalString("userLookupErrGid", "")
	userErrHome := jc.OptionalString("userLookupErrHome", "")

	// TODO: group? requires http://code.google.com/p/go/issues/detail?id=2617
	var runas *user.User
	if userStr != "" {
		runas, err = user.Lookup(userStr)
		if err != nil {
			if userErrUid != "" {
				runas = &user.User{
					Uid:      userErrUid,
					Gid:      userErrGid,
					Username: userStr,
					HomeDir:  userErrHome,
				}
			} else {
				return t.configError("%v", err)
			}
		}
		if stdEnv {
			env = append(env, fmt.Sprintf("USER=%s", userStr))
			env = append(env, fmt.Sprintf("HOME=%s", runas.HomeDir))
		}
	} else {
		if stdEnv {
			env = append(env, fmt.Sprintf("USER=%s", os.Getenv("USER")))
			env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
		}
	}

	envMap := jc.OptionalObject("env")
	envHas := func(k string) bool {
		_, ok := envMap[k]
		return ok
	}
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if stdEnv && !envHas("PATH") {
		env = append(env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/usr/sbin:/sbin:/bin")
	}

	extraFiles := []*os.File{}
	ports := jc.OptionalObject("ports")
	for portName, vi := range ports {
		var ln net.Listener
		var err error
		switch v := vi.(type) {
		case float64:
			ln, err = net.Listen("tcp", ":"+strconv.Itoa(int(v)))
		case string:
			ln, err = net.Listen("tcp", v)
		default:
			return t.configError("port %q value must be a string or integer", portName)
		}
		if err != nil {
			restartIn := 5 * time.Second
			time.AfterFunc(restartIn, func() {
				t.controlc <- updateMessage{t.tf}
			})
			return t.startError("port %q listen error: %v; restarting in %v", portName, err, restartIn)
		}
		lf, err := ln.(*net.TCPListener).File()
		if err != nil {
			return t.startError("error getting file of port %q listener: %v", portName, err)
		}
		logger.Printf("opened port named %q on %v; fd=%d", portName, vi, lf.Fd())
		ln.Close()
		env = append(env, fmt.Sprintf("RUNSIT_PORTFD_%s=%d", portName, 3+len(extraFiles)))
		extraFiles = append(extraFiles, lf)
		defer lf.Close()
	}

	bin := jc.RequiredString("binary")
	dir := jc.OptionalString("cwd", "")
	args := jc.OptionalList("args")
	groups := jc.OptionalList("groups")
	numFiles := jc.OptionalInt("numFiles", 0)
	if err := jc.Validate(); err != nil {
		return t.configError("configuration error: %v", err)
	}
	t.config = jc

	finalBin := bin
	if !filepath.IsAbs(bin) {
		dirAbs, err := filepath.Abs(dir)
		if err != nil {
			return t.configError("finding absolute path of dir %q: %v", dir, err)
		}
		finalBin = filepath.Clean(filepath.Join(dirAbs, bin))
	}

	_, err = os.Stat(finalBin)
	if err != nil {
		return t.configError("stat of binary %q failed: %v", bin, err)
	}

	argv := []string{filepath.Base(bin)}
	argv = append(argv, args...)

	lr := &LaunchRequest{
		Path:     bin,
		Env:      env,
		Dir:      dir,
		Argv:     argv,
		NumFiles: numFiles,
	}

	if runas != nil {
		lr.Uid = atoi(runas.Uid)
		lr.Gid = atoi(runas.Gid)
	}
	if groupStr != "" {
		gid, err := LookupGroupId(groupStr)
		if err != nil {
			return t.configError("error looking up group %q: %v", groupStr, err)
		}
		lr.Gid = gid // primary group
	}

	// supplemental groups:
	for _, group := range groups {
		gid, err := LookupGroupId(group)
		if err != nil {
			return t.configError("error looking up group %q: %v", group, err)
		}
		lr.Gids = append(lr.Gids, gid)
	}

	cmd, outPipe, errPipe, err := lr.start(extraFiles)
	if err != nil {
		return t.startError("failed to start: %v", err)
	}

	instance := &TaskInstance{
		task:      t,
		config:    jc,
		startTime: time.Now(),
		lr:        lr,
		cmd:       cmd,
	}

	t.Printf("started with PID %d", instance.Pid())
	t.running = instance
	go instance.watchPipe(outPipe, "stdout")
	go instance.watchPipe(errPipe, "stderr")
	go instance.awaitDeath()
	return nil
}

// run in its own goroutine
func (in *TaskInstance) awaitDeath() {
	in.waitErr = in.cmd.Wait()
	in.endTime = time.Now()
	in.task.controlc <- instanceGoneMessage{in}
}

// run in its own goroutine
func (in *TaskInstance) watchPipe(r io.Reader, name string) {
	br := bufio.NewReader(r)
	for {
		sl, isPrefix, err := br.ReadLine()
		if err == io.EOF {
			// Not worth logging about.
			return
		}
		if err != nil {
			in.Printf("pipe %q closed: %v", name, err)
			return
		}
		in.output.Add(&Line{
			T:        time.Now(),
			Name:     name,
			Data:     string(sl),
			isPrefix: isPrefix,
			instance: in,
		})
	}
	panic("unreachable")
}

func (t *Task) Stop() error {
	errc := make(chan error, 1)
	t.controlc <- stopMessage{errc}
	return <-errc
}

// runs in Task.loop
func (t *Task) stop() error {
	in := t.running
	if in == nil {
		return nil
	}

	// TODO: more graceful kill types
	in.Printf("sending SIGKILL")

	// Was: in.cmd.Process.Kill(); but we want to kill
	// the entire process group.
	processGroup := 0 - in.Pid()
	rv := syscall.Kill(processGroup, 9)
	in.Printf("Kill result: %v", rv)
	t.running = nil
	return nil
}

// TaskStatus is an one-time snapshot of a task's status, for rendering in
// the web UI.
type TaskStatus struct {
	Running  *TaskInstance   // or nil, if none running
	StartErr error           // if a task is not running, the reason why it failed to start
	ErrTime  time.Time       // time of StartErr
	StartIn  time.Duration   // non-zero if task is rate-limited and will restart in this time
	Failures []*TaskInstance // past few failures
}

func (s *TaskStatus) Summary() string {
	in := s.Running
	if in != nil {
		return "ok"
	}
	if err := s.StartErr; err != nil {
		return fmt.Sprintf("Start error (%v ago): %v", time.Now().Sub(s.ErrTime), err)
	}
	// TODO: flesh these not running states out.
	// e.g. intentionaly stopped, how long we're pausing before
	// next re-start attempt, etc.
	return "not running"
}

// Status returns the task's status.
func (t *Task) Status() *TaskStatus {
	ch := make(chan *TaskStatus, 1)
	t.controlc <- statusRequestMessage{resCh: ch}
	return <-ch
}

// runs in Task.loop
func (t *Task) status() *TaskStatus {
	failures := make([]*TaskInstance, len(t.failures))
	copy(failures, t.failures)
	s := &TaskStatus{
		Running:  t.running,
		Failures: failures,
	}
	if t.running == nil {
		s.StartErr = t.configErr
		s.ErrTime = t.errTime
	}
	return s
}

func watchConfigDir() {
	for tf := range dirWatcher().Updates() {
		t := GetOrMakeTask(tf.Name(), tf)
		go t.Update(tf)
	}
}

var (
	tasksMu sync.Mutex               // guards tasks
	tasks   = make(map[string]*Task) // name -> Task
)

func GetTask(name string) (t *Task, ok bool) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	t, ok = tasks[name]
	return
}

func DeleteTask(name string) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	delete(tasks, name)
}

// GetOrMakeTask returns or create the named task.
func GetOrMakeTask(name string, tf TaskFile) *Task {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	t, ok := tasks[name]
	if !ok {
		t = NewTask(name)
		t.tf = tf
		tasks[name] = t
	}
	return t
}

type byName []*Task

func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetTasks returns all known tasks.
func GetTasks() []*Task {
	ts := []*Task{}
	tasksMu.Lock()
	defer tasksMu.Unlock()
	for _, t := range tasks {
		ts = append(ts, t)
	}
	sort.Sort(byName(ts))
	return ts
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func handleSignals() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc)

	for s := range sigc {
		switch s {
		case os.Interrupt, os.Signal(syscall.SIGTERM):
			logger.Printf("Got signal %q; stopping all tasks.", s)
			for _, t := range GetTasks() {
				t.Stop()
			}
			logger.Printf("Tasks all stopped after %s; quitting.", s)
			os.Exit(0)
		case os.Signal(syscall.SIGCHLD):
			// Ignore.
		default:
			logger.Printf("unhandled signal: %T %#v", s, s)
		}
	}
}

func main() {
	MaybeBecomeChildProcess()
	flag.Parse()

	listenAddr := "localhost"
	if a := os.Getenv("RUNSIT_LISTEN"); a != "" {
		listenAddr = a
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenAddr, *httpPort))
	if err != nil {
		logger.Printf("Error listening on port %d: %v", *httpPort, err)
		os.Exit(1)
		return
	}
	logger.Printf("Listening on port %d", *httpPort)

	go handleSignals()
	go watchConfigDir()
	go runWebServer(ln)
	select {}
}
