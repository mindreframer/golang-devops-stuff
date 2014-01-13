package job_tracker

import (
	"bytes"
	"sync"

	"github.com/vito/garden/backend"
)

type namedStream struct {
	job     *Job
	name    string
	discard bool

	destination      *bytes.Buffer
	destinationMutex *sync.RWMutex
}

func newNamedStream(job *Job, name string, discard bool) *namedStream {
	return &namedStream{
		job:     job,
		name:    name,
		discard: discard,

		destination:      new(bytes.Buffer),
		destinationMutex: new(sync.RWMutex),
	}
}

func (s *namedStream) Write(data []byte) (int, error) {
	defer s.job.sendToStreams(backend.JobStream{
		Name: s.name,
		Data: data,
	})

	if s.discard {
		return len(data), nil
	}

	s.destinationMutex.Lock()
	defer s.destinationMutex.Unlock()

	return s.destination.Write(data)
}

func (s *namedStream) Bytes() []byte {
	s.destinationMutex.RLock()
	defer s.destinationMutex.RUnlock()

	return s.destination.Bytes()
}
