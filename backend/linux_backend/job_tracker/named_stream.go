package job_tracker

import (
	"io"
	"sync"

	"github.com/vito/garden/backend"
)

type namedStream struct {
	job         *Job
	name        string
	destination io.Writer

	sync.RWMutex
}

func newNamedStream(job *Job, name string, destination io.Writer) *namedStream {
	return &namedStream{
		job:         job,
		name:        name,
		destination: destination,
	}
}

func (s *namedStream) Write(data []byte) (int, error) {
	defer s.job.sendToStreams(backend.JobStream{
		Name: s.name,
		Data: data,
	})

	return s.destination.Write(data)
}
