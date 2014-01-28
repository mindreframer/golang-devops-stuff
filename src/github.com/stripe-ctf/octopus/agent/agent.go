package agent

import (
	"bufio"
	"fmt"
	"github.com/stripe-ctf/octopus/state"
	"github.com/stripe-ctf/octopus/unix"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// An agent.Agent is a process on the network that occasionally gets
// stopped and killed. Locally, processes are killed by giving them
// SIGTERM and stopped by giving them SIGTSTP. Remotely, processes are
// killed by killing their container, and stopped by freezing their
// container. (Some of the code for doing this is in a shim.)

type Agent struct {
	sync.Mutex
	ConnectionString, Name string
	dir, freezefile        string
	args                   []string

	// Refcounts for how many times someone has attempted to kill or stop
	// this agent.
	killcount, stopcount uint
	cmd                  *exec.Cmd
}

func copyLines(prefix string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("[%s] %s", prefix, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[%s] Error while reading: %v", prefix, err)
	}
}

func createCleanDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	return os.Mkdir(dir, os.ModeDir|0755)
}

func NewAgent(i uint) *Agent {
	args := make([]string, 0)
	var container string

	if !state.Local() {
		// Run under LXC remotely
		homedir := fmt.Sprintf("/home/%s", state.Username())
		container = state.ContainerIds()[i]
		args = append(args,
			"lxc-attach", "-n", container,
			"--clear-env", "--",
			"/ctf3/shell.sh", state.Username(), homedir,
		)
	}
	args = append(args, state.Sqlcluster())
	args = append(args,
		"-l", SocketName(i),
		"-d", ContainerWorkingDir(i))
	if i != 0 {
		args = append(args, "--join="+SocketName(0))
	}
	args = append(args, state.Args()...)

	cs, err := unix.Encode(SocketPath(i, i))
	if err != nil {
		log.Fatalf("Couldn't create node %d: %s", i, err)
	}

	return &Agent{
		Name:             NodeName(i),
		ConnectionString: cs,
		args:             args,
		dir:              WorkingDir(i),
		// This is going to contain nonsense if container is empty, but
		// it won't matter
		freezefile: "/sys/fs/cgroup/freezer/lxc/" + container +
			"/freezer.state",
	}
}

func (a *Agent) Prepare() {
	if state.Local() {
		err := createCleanDir(a.dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Going to run %s as %s", a.Name, a.args)
}

func (a *Agent) Dryrun() {
	if _, err := os.Stat(a.dir); !os.IsNotExist(err) {
		log.Printf("Would clear: %s", a.dir)
	}

	log.Printf("Would have run: %s", a.args)
}

func (a *Agent) Start() {
	go func() {
		state.WaitGroup().Wait()
		if a.cmd != nil && a.cmd.Process != nil {
			a.cmd.Process.Signal(syscall.SIGTERM)
			a.cmd = nil
		}
		state.WaitGroup().Done()
	}()
	a.Lock()
	defer a.Unlock()
	if a.cmd == nil {
		a.start()
	}
}

func (a *Agent) start() {
	if a.cmd != nil {
		log.Fatal("Cannot call start(): program already running!")
	}
	if a.killcount > 0 {
		return
	}
	cmd := exec.Command(a.args[0], a.args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Unable to open pipe to stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal("Unable to open pipe to stderr: %v", err)
	}
	if state.Local() {
		cmd.Dir = a.dir
	}
	go copyLines(a.Name, stdout)
	go copyLines(a.Name, stderr)
	log.Printf("Starting %v", a)
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Could't spawn %v: %s", a, err)
	}
	go func() {
		err := cmd.Wait()
		a.Lock()
		defer a.Unlock()

		// Not the active command
		if a.cmd != cmd {
			return
		}

		var d string
		if err != nil {
			d = fmt.Sprintf("Command exited unexpectedly: %s (%s)", cmd, err)
		} else {
			d = fmt.Sprintf("Command exited unexpectedly (but cleanly!): %s", cmd)
		}
		state.RecordDisqualifier(d)

		state.WaitGroup().Exit()
	}()

	a.cmd = cmd
}

func (a *Agent) Kill(dt time.Duration) {
	a.Lock()
	if a.killcount == 0 && a.cmd != nil {
		a.cmd.Process.Signal(syscall.SIGTERM)
		a.cmd = nil
	}
	a.killcount++
	a.Unlock()

	time.Sleep(dt)

	a.Lock()
	defer a.Unlock()
	a.killcount--
	// If an agent is both killed and stopped, we leave it in a killed state
	// as opposed to launching-and-immediately-stopping
	if a.killcount == 0 && a.stopcount == 0 {
		a.start()
	}
}

func (a *Agent) Stop(dt time.Duration) {
	a.Freeze()
	time.Sleep(dt)
	a.Thaw()
}

func (a *Agent) Freeze() {
	a.Lock()
	defer a.Unlock()

	if a.stopcount == 0 && a.cmd != nil {
		if state.Local() {
			a.cmd.Process.Signal(syscall.SIGTSTP)
		} else {
			file, err := os.Open(a.freezefile)
			if err != nil {
				log.Printf("freeze error: %v", err)
				return
			}
			file.WriteString("FROZEN\n")
			file.Close()
		}
	}

	a.stopcount++
}

func (a *Agent) Thaw() {
	a.Lock()
	defer a.Unlock()

	a.stopcount--

	if a.stopcount == 0 && a.cmd == nil {
		a.start()
	} else if a.stopcount == 0 {
		if state.Local() {
			a.cmd.Process.Signal(syscall.SIGCONT)
		} else {
			file, err := os.Open(a.freezefile)
			if err != nil {
				log.Printf("thaw error: %v", err)
				return
			}
			file.WriteString("THAWED\n")
			file.Close()
		}
	}
}

func (a *Agent) String() string {
	return fmt.Sprintf("Agent<%v>", a.Name)
}
