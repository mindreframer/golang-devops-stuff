package job_tracker

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/command_runner"
)

type JobTracker struct {
	containerPath string
	runner        command_runner.CommandRunner

	jobs      map[uint32]*Job
	nextJobID uint32

	sync.RWMutex
}

type UnknownJobError struct {
	JobID uint32
}

func (e UnknownJobError) Error() string {
	return fmt.Sprintf("unknown job: %d", e.JobID)
}

func New(containerPath string, runner command_runner.CommandRunner) *JobTracker {
	return &JobTracker{
		containerPath: containerPath,
		runner:        runner,

		jobs: make(map[uint32]*Job),
	}
}

func (t *JobTracker) Spawn(cmd *exec.Cmd, discardOutput bool) (uint32, error) {
	t.Lock()

	jobID := t.nextJobID
	t.nextJobID++

	job := NewJob(jobID, discardOutput, t.containerPath, cmd, t.runner)

	t.jobs[jobID] = job

	t.Unlock()

	ready, active := job.Spawn()

	err := <-ready
	if err != nil {
		return 0, err
	}

	go t.Link(jobID)

	err = <-active
	if err != nil {
		return 0, err
	}

	return jobID, nil
}

func (t *JobTracker) Link(jobID uint32) (uint32, []byte, []byte, error) {
	t.RLock()
	job, ok := t.jobs[jobID]
	t.RUnlock()

	if !ok {
		return 0, nil, nil, UnknownJobError{jobID}
	}

	defer t.unregister(jobID)

	return job.Link()
}

func (t *JobTracker) Stream(jobID uint32) (chan backend.JobStream, error) {
	t.RLock()
	job, ok := t.jobs[jobID]
	t.RUnlock()

	if !ok {
		return nil, UnknownJobError{jobID}
	}

	return job.Stream(), nil
}

func (t *JobTracker) ActiveJobs() []uint32 {
	jobs := []uint32{}

	for id, _ := range t.jobs {
		jobs = append(jobs, id)
	}

	return jobs
}

func (t *JobTracker) unregister(jobID uint32) {
	t.Lock()
	defer t.Unlock()

	delete(t.jobs, jobID)
}
