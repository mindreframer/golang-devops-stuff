package job_tracker

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/command_runner"
)

type Job struct {
	id            uint32
	discardOutput bool
	containerPath string
	cmd           *exec.Cmd
	runner        command_runner.CommandRunner

	waitingLinks *sync.Cond
	runningLink  *sync.Once

	streams    []chan backend.JobStream
	streamLock *sync.RWMutex

	completed bool

	exitStatus uint32
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
}

func NewJob(
	id uint32,
	discardOutput bool,
	containerPath string,
	cmd *exec.Cmd,
	runner command_runner.CommandRunner,
) *Job {
	return &Job{
		id:            id,
		discardOutput: discardOutput,
		containerPath: containerPath,
		cmd:           cmd,
		runner:        runner,

		waitingLinks: sync.NewCond(&sync.Mutex{}),
		runningLink:  &sync.Once{},
		streamLock:   &sync.RWMutex{},
	}
}

func (j *Job) Spawn() (ready, active chan error) {
	ready = make(chan error, 1)
	active = make(chan error, 1)

	spawnPath := path.Join(j.containerPath, "bin", "iomux-spawn")
	jobDir := path.Join(j.containerPath, "jobs", fmt.Sprintf("%d", j.id))

	mkdir := &exec.Cmd{
		Path: "mkdir",
		Args: []string{"-p", jobDir},
	}

	err := j.runner.Run(mkdir)
	if err != nil {
		ready <- err
		return
	}

	spawn := &exec.Cmd{
		Path:  spawnPath,
		Stdin: j.cmd.Stdin,
	}

	spawn.Args = append([]string{jobDir}, j.cmd.Path)
	spawn.Args = append(spawn.Args, j.cmd.Args...)

	spawnR, spawnW, err := os.Pipe()
	if err != nil {
		ready <- err
		return
	}

	spawn.Stdout = spawnW

	spawnOut := bufio.NewReader(spawnR)

	err = j.runner.Start(spawn)
	if err != nil {
		ready <- err
		return
	}

	go func() {
		defer func() {
			spawn.Wait()
			spawnW.Close()
			spawnR.Close()
		}()

		_, err = spawnOut.ReadBytes('\n')
		if err != nil {
			ready <- err
			return
		}

		ready <- nil

		_, err = spawnOut.ReadBytes('\n')
		if err != nil {
			active <- err
			return
		}

		active <- nil
	}()

	return
}

func (j *Job) Link() (uint32, []byte, []byte, error) {
	j.waitingLinks.L.Lock()
	defer j.waitingLinks.L.Unlock()

	if j.completed {
		return j.exitStatus, j.stdout.Bytes(), j.stderr.Bytes(), nil
	}

	j.runningLink.Do(j.runLinker)

	if !j.completed {
		j.waitingLinks.Wait()
	}

	return j.exitStatus, j.stdout.Bytes(), j.stderr.Bytes(), nil
}

func (j *Job) Stream() chan backend.JobStream {
	return j.registerStream()
}

func (j *Job) runLinker() {
	linkPath := path.Join(j.containerPath, "bin", "iomux-link")
	jobDir := path.Join(j.containerPath, "jobs", fmt.Sprintf("%d", j.id))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	var cmdStdout, cmdStderr io.Writer

	if j.discardOutput {
		cmdStdout = ioutil.Discard
		cmdStderr = ioutil.Discard
	} else {
		cmdStdout = stdout
		cmdStderr = stderr
	}

	link := &exec.Cmd{
		Path:   linkPath,
		Args:   []string{jobDir},
		Stdout: newNamedStream(j, "stdout", cmdStdout),
		Stderr: newNamedStream(j, "stderr", cmdStderr),
	}

	j.runner.Run(link)

	exitStatus := uint32(255)

	if link.ProcessState != nil {
		// TODO: why do I need to modulo this?
		exitStatus = uint32(link.ProcessState.Sys().(syscall.WaitStatus) % 255)
	}

	j.exitStatus = exitStatus
	j.stdout = stdout
	j.stderr = stderr

	j.completed = true

	j.sendToStreams(backend.JobStream{ExitStatus: &exitStatus})
	j.closeStreams()

	j.waitingLinks.Broadcast()
}

func (j *Job) registerStream() chan backend.JobStream {
	j.streamLock.Lock()
	defer j.streamLock.Unlock()

	stream := make(chan backend.JobStream)

	j.streams = append(j.streams, stream)

	return stream
}

func (j *Job) sendToStreams(chunk backend.JobStream) {
	j.streamLock.RLock()
	defer j.streamLock.RUnlock()

	for _, sink := range j.streams {
		sink <- chunk
	}
}

func (j *Job) closeStreams() {
	j.streamLock.RLock()
	defer j.streamLock.RUnlock()

	for _, sink := range j.streams {
		close(sink)
	}
}
