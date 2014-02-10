package job_tracker

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/command_runner"
)

type Job struct {
	ID            uint32
	DiscardOutput bool

	containerPath string
	runner        command_runner.CommandRunner

	waitingLinks *sync.Cond
	runningLink  *sync.Once
	link         *exec.Cmd
	unlinked     bool

	streams      chan backend.JobStream
	closeStreams chan bool
	addStream    chan chan backend.JobStream

	completed bool

	exitStatus uint32
	stdout     *namedStream
	stderr     *namedStream
}

func NewJob(
	id uint32,
	discardOutput bool,
	containerPath string,
	runner command_runner.CommandRunner,
) *Job {
	j := &Job{
		ID:            id,
		DiscardOutput: discardOutput,

		containerPath: containerPath,
		runner:        runner,

		streams:      make(chan backend.JobStream),
		closeStreams: make(chan bool),
		addStream:    make(chan chan backend.JobStream),

		waitingLinks: sync.NewCond(&sync.Mutex{}),
		runningLink:  &sync.Once{},
	}

	j.stdout = newNamedStream(j, "stdout", j.DiscardOutput)
	j.stderr = newNamedStream(j, "stderr", j.DiscardOutput)

	go j.dispatchStreams()

	return j
}

func (j *Job) Spawn(cmd *exec.Cmd) (ready, active chan error) {
	ready = make(chan error, 1)
	active = make(chan error, 1)

	spawnPath := path.Join(j.containerPath, "bin", "iomux-spawn")
	jobDir := path.Join(j.containerPath, "jobs", fmt.Sprintf("%d", j.ID))

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
		Stdin: cmd.Stdin,
	}

	spawn.Args = append([]string{jobDir}, cmd.Path)
	spawn.Args = append(spawn.Args, cmd.Args...)

	spawn.Env = cmd.Env

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

func (j *Job) Unlink() error {
	if j.link != nil {
		j.unlinked = true
		return j.runner.Signal(j.link, os.Interrupt)
	}

	return nil
}

func (j *Job) Stream() chan backend.JobStream {
	return j.registerStream()
}

func (j *Job) runLinker() {
	linkPath := path.Join(j.containerPath, "bin", "iomux-link")
	jobDir := path.Join(j.containerPath, "jobs", fmt.Sprintf("%d", j.ID))

	j.link = &exec.Cmd{
		Path:   linkPath,
		Args:   []string{"-w", path.Join(jobDir, "cursors"), jobDir},
		Stdout: j.stdout,
		Stderr: j.stderr,
	}

	j.runner.Run(j.link)

	if j.unlinked {
		// iomux-link was killed on shutdown via .Unlink; command didn't
		// actually exit, so just block forever until server dies and re-links
		select {}
	}

	exitStatus := uint32(255)

	if j.link.ProcessState != nil {
		exitStatus = uint32(j.link.ProcessState.Sys().(syscall.WaitStatus).ExitStatus())
	}

	j.exitStatus = exitStatus

	j.completed = true

	j.sendToStreams(backend.JobStream{ExitStatus: &exitStatus})
	j.closeStreams <- true

	j.waitingLinks.Broadcast()
}

func (j *Job) registerStream() chan backend.JobStream {
	stream := make(chan backend.JobStream, 2)

	stdout := j.stdout.Bytes()
	stderr := j.stderr.Bytes()

	if len(stdout) > 0 {
		stream <- backend.JobStream{
			Name: "stdout",
			Data: stdout,
		}
	}

	if len(stderr) > 0 {
		stream <- backend.JobStream{
			Name: "stderr",
			Data: stderr,
		}
	}

	j.addStream <- stream

	return stream
}

func (j *Job) sendToStreams(chunk backend.JobStream) {
	j.streams <- chunk
}

func (j *Job) dispatchStreams() {
	streams := []chan backend.JobStream{}

	for {
		select {
		case stream := <-j.addStream:
			streams = append(streams, stream)

		case chunk := <-j.streams:
			for _, stream := range streams {
				stream <- chunk
			}

		case <-j.closeStreams:
			for _, stream := range streams {
				close(stream)
			}

			return
		}
	}
}
