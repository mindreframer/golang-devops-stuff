package fake_gordon

import (
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/nu7hatch/gouuid"
	"github.com/vito/gordon/warden"
	"sync"
)

type FakeGordon struct {
	Connected    bool
	ConnectError error

	createdHandles []string
	CreateError    error

	StopError error

	destroyedHandles []string
	DestroyError     error

	SpawnError error

	LinkError error

	NetInError error

	LimitMemoryError error

	GetMemoryLimitError error

	LimitDiskError error

	GetDiskLimitError error

	ListError error

	InfoError error

	CopyInError error

	StreamError error

	scriptsThatRan      []*RunningScript
	runCallbacks        map[*RunningScript]RunCallback
	runReturnStatusCode uint32
	runReturnError      error

	lock *sync.Mutex
}

type RunCallback func() (*warden.RunResponse, error)

type RunningScript struct {
	Handle string
	Script string
}

func New() *FakeGordon {
	f := &FakeGordon{}
	f.Reset()
	return f
}

func (f *FakeGordon) Reset() {
	f.lock = &sync.Mutex{}
	f.Connected = false
	f.ConnectError = nil

	f.createdHandles = []string{}
	f.CreateError = nil

	f.StopError = nil

	f.destroyedHandles = []string{}
	f.DestroyError = nil

	f.SpawnError = nil
	f.LinkError = nil
	f.NetInError = nil
	f.LimitMemoryError = nil
	f.GetMemoryLimitError = nil
	f.LimitDiskError = nil
	f.GetDiskLimitError = nil
	f.ListError = nil
	f.InfoError = nil
	f.CopyInError = nil
	f.StreamError = nil

	f.scriptsThatRan = make([]*RunningScript, 0)
	f.runCallbacks = make(map[*RunningScript]RunCallback)
	f.runReturnStatusCode = 0
	f.runReturnError = nil
}

func (f *FakeGordon) Connect() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.Connected = true
	return f.ConnectError
}

func (f *FakeGordon) Create() (*warden.CreateResponse, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.CreateError != nil {
		return nil, f.CreateError
	}

	handleUuid, _ := uuid.NewV4()
	handle := handleUuid.String()[:11]

	f.createdHandles = append(f.createdHandles, handle)

	return &warden.CreateResponse{
		Handle: proto.String(handle),
	}, nil
}

func (f *FakeGordon) CreatedHandles() []string {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.createdHandles
}

func (f *FakeGordon) Stop(handle string, background, kill bool) (*warden.StopResponse, error) {
	panic("NOOP!")
	return nil, f.StopError
}

func (f *FakeGordon) Destroy(handle string) (*warden.DestroyResponse, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.DestroyError != nil {
		return nil, f.DestroyError
	}

	f.destroyedHandles = append(f.destroyedHandles, handle)

	return &warden.DestroyResponse{}, nil
}

func (f *FakeGordon) DestroyedHandles() []string {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.destroyedHandles
}

func (f *FakeGordon) Spawn(handle, script string, discardOutput bool) (*warden.SpawnResponse, error) {
	panic("NOOP!")
	return nil, f.SpawnError
}

func (f *FakeGordon) Link(handle string, jobID uint32) (*warden.LinkResponse, error) {
	panic("NOOP!")
	return nil, f.LinkError
}

func (f *FakeGordon) NetIn(handle string) (*warden.NetInResponse, error) {
	panic("NOOP!")
	return nil, f.NetInError
}

func (f *FakeGordon) LimitMemory(handle string, limit uint64) (*warden.LimitMemoryResponse, error) {
	panic("NOOP!")
	return nil, f.LimitMemoryError
}

func (f *FakeGordon) GetMemoryLimit(handle string) (uint64, error) {
	panic("NOOP!")
	return 0, f.GetMemoryLimitError
}

func (f *FakeGordon) LimitDisk(handle string, limit uint64) (*warden.LimitDiskResponse, error) {
	panic("NOOP!")
	return nil, f.LimitDiskError
}

func (f *FakeGordon) GetDiskLimit(handle string) (uint64, error) {
	panic("NOOP!")
	return 0, f.GetDiskLimitError
}

func (f *FakeGordon) List() (*warden.ListResponse, error) {
	panic("NOOP!")
	return nil, f.ListError
}

func (f *FakeGordon) Info(handle string) (*warden.InfoResponse, error) {
	panic("NOOP!")
	return nil, f.InfoError
}

func (f *FakeGordon) CopyIn(handle, src, dst string) (*warden.CopyInResponse, error) {
	panic("NOOP!")
	return nil, f.CopyInError
}

func (f *FakeGordon) Stream(handle string, jobID uint32) (<-chan *warden.StreamResponse, error) {
	panic("NOOP!")
	return nil, f.StreamError
}

func (f *FakeGordon) ScriptsThatRan() []*RunningScript {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.scriptsThatRan
}

func (f *FakeGordon) SetRunReturnValues(statusCode uint32, err error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.runReturnStatusCode = statusCode
	f.runReturnError = err
}

func (f *FakeGordon) WhenRunning(handle string, script string, callback RunCallback) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.runCallbacks[&RunningScript{handle, script}] = callback
}

func (f *FakeGordon) Run(handle string, script string) (*warden.RunResponse, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.scriptsThatRan = append(f.scriptsThatRan, &RunningScript{
		Handle: handle,
		Script: script,
	})

	for ro, cb := range f.runCallbacks {
		if ro.Handle == handle && ro.Script == script {
			return cb()
		}
	}

	return &warden.RunResponse{ExitStatus: proto.Uint32(f.runReturnStatusCode)}, f.runReturnError
}
