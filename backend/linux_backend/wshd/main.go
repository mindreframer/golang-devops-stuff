// +build linux

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/vito/garden/backend/linux_backend/wshd/barrier"
	"github.com/vito/garden/backend/linux_backend/wshd/daemon"
	"github.com/vito/garden/command_runner"
)

var runPath = flag.String(
	"run",
	"./run",
	"where to put the gnome daemon .sock file",
)

var rootPath = flag.String(
	"root",
	"./root",
	"root filesystem for the container",
)

var libPath = flag.String(
	"lib",
	"./lib",
	"directory containing hooks",
)

var title = flag.String(
	"title",
	"garden gnome",
	"title for the container gnome daemon",
)

var continueAsChild = flag.Bool(
	"continue",
	false,
	"(internal) continue execution as containerized daemon",
)

func main() {
	flag.Parse()

	if *continueAsChild {
		childContinue()
		return
	}

	commandRunner := command_runner.New(true)

	fullRunPath := resolvePath(*runPath)
	fullLibPath := resolvePath(*libPath)
	fullRootPath := resolvePath(*rootPath)

	runtime.LockOSThread()

	err := syscall.Unshare(CLONE_NEWNS)
	if err != nil {
		log.Fatalln("error unsharing:", err)
	}

	err = commandRunner.Run(&exec.Cmd{Path: path.Join(fullLibPath, "hook-parent-before-clone.sh")})
	if err != nil {
		log.Fatalln("error executing hook-parent-before-clone.sh:", err)
	}

	listener, err := net.Listen("unix", path.Join(fullRunPath, "wshd.sock"))
	if err != nil {
		log.Fatalln("error listening:", err)
	}

	socketFile, err := listener.(*net.UnixListener).File()
	if err != nil {
		log.Fatalln("error getting listening file:", err)
	}

	newFD, err := syscall.Dup(int(socketFile.Fd()))
	if err != nil {
		log.Fatalln("error duplicating socket file:", err)
	}

	logFile, err := os.Create(path.Join(fullRunPath, "wshd.log"))
	if err != nil {
		log.Fatalln("error creating wshd log file:", err)
	}

	logFD, err := syscall.Dup(int(logFile.Fd()))
	if err != nil {
		log.Fatalln("error duplicating log file:", err)
	}

	parentBarrier, err := barrier.New()
	if err != nil {
		log.Fatalln("error creating parent barrier:", err)
	}

	childBarrier, err := barrier.New()
	if err != nil {
		log.Fatalln("error creating child barrier:", err)
	}

	state := State{
		SocketFD:      newFD,
		LogFD:         logFD,
		ChildBarrier:  childBarrier,
		ParentBarrier: parentBarrier,
	}

	pid, err := createContainerizedProcess()
	if err != nil {
		log.Fatalln("error creating child process:", err)
	}

	if pid == 0 {
		childRun(state, fullLibPath, fullRootPath, commandRunner)

		panic("unreachable")
	}

	os.Setenv("PID", fmt.Sprintf("%d", pid))

	err = commandRunner.Run(&exec.Cmd{Path: path.Join(fullLibPath, "hook-parent-after-clone.sh")})
	if err != nil {
		log.Fatalln("error executing hook-parent-after-clone.sh:", err)
	}

	err = parentBarrier.Signal()
	if err != nil {
		log.Fatalln("error signaling child:", err)
	}

	err = childBarrier.Wait()
	if err != nil {
		log.Fatalln("error waiting for signal from child:", err)
	}

	os.Exit(0)
}

func resolvePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalln("error resolving path:", path, err)
	}

	fullPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		log.Fatalln("error resolving path:", path, err)
	}

	return fullPath
}

func createContainerizedProcess() (int, error) {
	syscall.ForkLock.Lock()
	defer syscall.ForkLock.Unlock()

	var flags uintptr

	flags |= CLONE_NEWNS  // new mount namespace
	flags |= CLONE_NEWUTS // new UTS namespace
	flags |= CLONE_NEWIPC // new inter-process communiucation namespace
	flags |= CLONE_NEWPID // new process pid namespace
	flags |= CLONE_NEWNET // new network namespace

	pid, _, err := syscall.RawSyscall(syscall.SYS_CLONE, flags, 0, 0)
	if err != 0 {
		return 0, err
	}

	return int(pid), nil
}

func childRun(state State, fullLibPath, fullRootPath string, commandRunner command_runner.CommandRunner) {
	err := state.ParentBarrier.Wait()
	if err != nil {
		log.Println("error waiting for signal from parent:", err)
		goto cleanup
	}

	err = commandRunner.Run(&exec.Cmd{Path: path.Join(fullLibPath, "hook-child-before-pivot.sh")})
	if err != nil {
		log.Println("error executing hook-child-before-pivot.sh:", err)
		goto cleanup
	}

	err = os.Chdir(fullRootPath)
	if err != nil {
		log.Println("error chdir'ing to root path:", err)
		goto cleanup
	}

	err = os.MkdirAll("mnt", 0700)
	if err != nil {
		log.Println("error making mnt:", err)
		goto cleanup
	}

	err = syscall.PivotRoot(".", "mnt")
	if err != nil {
		log.Println("error pivoting root:", err)
		goto cleanup
	}

	err = os.Chdir("/")
	if err != nil {
		log.Println("error chdir'ing to /:", err)
		goto cleanup
	}

	err = commandRunner.Run(&exec.Cmd{Path: path.Join("/mnt", fullLibPath, "hook-child-after-pivot.sh")})
	if err != nil {
		log.Println("error executing hook-child-after-pivot.sh:", err)
		goto cleanup
	}

	err = SaveStateToSHM(state)
	if err != nil {
		log.Println("error saving state:", err)
		goto cleanup
	}

	err = syscall.Exec("/sbin/wshd", []string{*title, "--continue"}, []string{})
	if err != nil {
		log.Println("error executing /sbin/wshd:", err)
		goto cleanup
	}
cleanup:
	state.ChildBarrier.Fail()
	os.Exit(255)
}

func childContinue() {
	state, err := LoadStateFromSHM()
	if err != nil {
		log.Fatalln("error loading state:", err)
	}

	syscall.CloseOnExec(state.ChildBarrier.FDs[0])
	syscall.CloseOnExec(state.ChildBarrier.FDs[1])
	syscall.CloseOnExec(state.SocketFD)
	syscall.CloseOnExec(state.LogFD)

	logFile := os.NewFile(uintptr(state.LogFD), "wshd.log")

	log.SetOutput(logFile)

	err = umountAll("/mnt")
	if err != nil {
		log.Fatalf("error clearing /mnt mountpoints: %#v\n", err)
	}

	_, err = syscall.Setsid()
	if err != nil {
		log.Fatalln("error setting sid:", err)
	}

	socketFile := os.NewFile(uintptr(state.SocketFD), "wshd.sock")

	daemon := daemon.New(socketFile)

	err = state.ChildBarrier.Signal()
	if err != nil {
		log.Fatalln("error signalling parent:", err)
	}

	err = daemon.Start()
	if err != nil {
		log.Fatalln("daemon start error:", err)
	}

	socketFile.Close()

	os.Stdin.Close()
	os.Stdout.Close()
	os.Stderr.Close()

	select {}
}
