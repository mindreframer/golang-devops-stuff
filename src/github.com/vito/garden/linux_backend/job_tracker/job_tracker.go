package job_tracker

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/command_runner"
)

type JobTracker struct {
	containerPath string
	runner        command_runner.CommandRunner

	jobs      map[uint32]*Job
	nextJobID uint32
	jobsMutex *sync.RWMutex
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

		jobs:      make(map[uint32]*Job),
		jobsMutex: new(sync.RWMutex),
	}
}

func (t *JobTracker) Spawn(cmd *exec.Cmd, discardOutput, autoLink bool) (uint32, error) {
	t.jobsMutex.Lock()

	jobID := t.nextJobID
	t.nextJobID++

	job := NewJob(jobID, discardOutput, t.containerPath, t.runner)

	t.jobs[jobID] = job

	t.jobsMutex.Unlock()

	ready, active := job.Spawn(cmd)

	err := <-ready
	if err != nil {
		return 0, err
	}

	if autoLink {
		go t.Link(jobID)

		err = <-active
		if err != nil {
			return 0, err
		}
	}

	return jobID, nil
}

func (t *JobTracker) Restore(jobID uint32, discardOutput bool) {
	t.jobsMutex.Lock()

	job := NewJob(jobID, discardOutput, t.containerPath, t.runner)

	t.jobs[jobID] = job

	if jobID >= t.nextJobID {
		t.nextJobID = jobID + 1
	}

	t.jobsMutex.Unlock()

	go t.Link(jobID)
}

func (t *JobTracker) Link(jobID uint32) (uint32, []byte, []byte, error) {
	t.jobsMutex.RLock()
	job, ok := t.jobs[jobID]
	t.jobsMutex.RUnlock()

	if !ok {
		return 0, nil, nil, UnknownJobError{jobID}
	}

	defer t.unregister(jobID)

	return job.Link()
}

func (t *JobTracker) Stream(jobID uint32) (chan backend.JobStream, error) {
	t.jobsMutex.RLock()
	job, ok := t.jobs[jobID]
	t.jobsMutex.RUnlock()

	if !ok {
		return nil, UnknownJobError{jobID}
	}

	go t.Link(jobID)

	return job.Stream(), nil
}

func (t *JobTracker) ActiveJobs() []*Job {
	t.jobsMutex.RLock()
	defer t.jobsMutex.RUnlock()

	jobs := []*Job{}

	for _, job := range t.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (t *JobTracker) unregister(jobID uint32) {
	t.jobsMutex.Lock()
	defer t.jobsMutex.Unlock()

	delete(t.jobs, jobID)
}
