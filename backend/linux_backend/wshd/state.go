package main

import (
	"bytes"
	"encoding/gob"
	"syscall"
	"unsafe"

	"github.com/vito/garden/backend/linux_backend/wshd/barrier"
)

type State struct {
	SocketFD      int
	LogFD         int
	ChildBarrier  *barrier.Barrier
	ParentBarrier *barrier.Barrier
}

const StateKey = 0xdeadbeef
const MaxStateSize = 1024

func SaveStateToSHM(state State) error {
	stateEncoded := new(bytes.Buffer)

	encoder := gob.NewEncoder(stateEncoded)
	err := encoder.Encode(state)
	if err != nil {
		return err
	}

	buf := stateEncoded.Bytes()

	shmid, _, errno := syscall.RawSyscall(syscall.SYS_SHMGET, StateKey, MaxStateSize, IPC_CREAT|IPC_EXCL|0600)
	if errno != 0 {
		return errno
	}

	addr, _, errno := syscall.RawSyscall(syscall.SYS_SHMAT, shmid, 0, 0)
	if errno != 0 {
		return errno
	}

	for i := uintptr(0); i < uintptr(len(buf)); i++ {
		*(*byte)(unsafe.Pointer(addr + i)) = buf[i]
	}

	return nil
}

func LoadStateFromSHM() (State, error) {
	var state State

	shmid, _, errno := syscall.RawSyscall(syscall.SYS_SHMGET, StateKey, MaxStateSize, 0600)
	if errno != 0 {
		return State{}, errno
	}

	addr, _, errno := syscall.RawSyscall(syscall.SYS_SHMAT, shmid, 0, 0)
	if errno != 0 {
		return State{}, errno
	}

	stateEncoded := make([]byte, MaxStateSize)

	copy(stateEncoded, (*[MaxStateSize]byte)(unsafe.Pointer(addr))[:])

	decoder := gob.NewDecoder(bytes.NewBuffer(stateEncoded))

	err := decoder.Decode(&state)
	if err != nil {
		return State{}, err
	}

	_, _, errno = syscall.RawSyscall(syscall.SYS_SHMDT, addr, 0, 0)
	if errno != 0 {
		return State{}, errno
	}

	_, _, errno = syscall.RawSyscall(syscall.SYS_SHMCTL, shmid, IPC_RMID, 0)
	if errno != 0 {
		return State{}, errno
	}

	return state, err
}
