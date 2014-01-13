package fake_port_pool

type FakePortPool struct {
	nextPort uint32

	AcquireError error
	RemoveError  error

	Acquired []uint32
	Released []uint32
	Removed  []uint32
}

func New(start uint32) *FakePortPool {
	return &FakePortPool{
		nextPort: start,
	}
}

func (p *FakePortPool) Acquire() (uint32, error) {
	if p.AcquireError != nil {
		return 0, p.AcquireError
	}

	port := p.nextPort
	p.nextPort++

	return port, nil
}

func (p *FakePortPool) Remove(port uint32) error {
	if p.RemoveError != nil {
		return p.RemoveError
	}

	p.Removed = append(p.Removed, port)

	return nil
}

func (p *FakePortPool) Release(port uint32) {
	p.Released = append(p.Released, port)
}
