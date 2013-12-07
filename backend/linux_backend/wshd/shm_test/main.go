package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const StateKey = 0xdeafbeef
const StateSize = 1

const (
	IPC_RMID  = 0
	IPC_CREAT = 1 << 9
	IPC_EXCL  = 2 << 9
)

func main() {
	shmid, _, errno := syscall.RawSyscall(syscall.SYS_SHMGET, StateKey, StateSize, IPC_CREAT|IPC_EXCL|0600)
	if errno != 0 {
		os.Exit(1)
		return
	}

	addr, _, errno := syscall.RawSyscall(syscall.SYS_SHMAT, shmid, 0, 0)
	if errno != 0 {
		os.Exit(2)
		return
	}

	sig := make(chan os.Signal)

	signal.Notify(sig, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sig

		fmt.Println("releasing")

		_, _, errno = syscall.RawSyscall(syscall.SYS_SHMDT, addr, 0, 0)
		if errno != 0 {
			os.Exit(3)
			return
		}

		_, _, errno = syscall.RawSyscall(syscall.SYS_SHMCTL, shmid, IPC_RMID, 0)
		if errno != 0 {
			os.Exit(4)
			return
		}

		os.Exit(0)
	}()

	fmt.Println("ok")

	select {}
}
