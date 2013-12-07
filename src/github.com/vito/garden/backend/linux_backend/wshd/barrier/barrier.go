package barrier

import (
	"fmt"
	"syscall"
)

type Barrier struct {
	FDs []int
}

func New() (*Barrier, error) {
	fds := make([]int, 2)

	err := syscall.Pipe(fds)
	if err != nil {
		return nil, err
	}

	return &Barrier{fds}, nil
}

func (b *Barrier) Wait() error {
	buf := make([]byte, 1)

	b.closeSignal()
	defer b.closeWait()

	_, err := syscall.Read(b.FDs[0], buf)
	if err != nil {
		return err
	}

	if buf[0] != 0 {
		return fmt.Errorf("barrier failed")
	}

	return nil
}

func (b *Barrier) Signal() error {
	return b.write(0)
}

func (b *Barrier) Fail() error {
	return b.write(1)
}

func (b *Barrier) write(datum byte) error {
	b.closeWait()
	defer b.closeSignal()

	_, err := syscall.Write(b.FDs[1], []byte{datum})
	return err
}

func (b *Barrier) closeWait() error {
	return syscall.Close(b.FDs[0])
}

func (b *Barrier) closeSignal() error {
	return syscall.Close(b.FDs[1])
}
