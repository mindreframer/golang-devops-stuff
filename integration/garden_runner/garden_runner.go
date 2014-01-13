package garden_runner

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vito/cmdtest"
)

type GardenRunner struct {
	SocketPath    string
	DepotPath     string
	RootPath      string
	RootFSPath    string
	SnapshotsPath string

	gardenBin string
	gardenCmd *exec.Cmd

	tmpdir string
}

func New(rootPath, rootFSPath string) (*GardenRunner, error) {
	gardenBin, err := cmdtest.Build("github.com/vito/garden")
	if err != nil {
		return nil, err
	}

	tmpdir, err := ioutil.TempDir(os.TempDir(), "garden-runner")
	if err != nil {
		return nil, err
	}

	depotPath := filepath.Join(tmpdir, "containers")

	err = os.Mkdir(depotPath, 0755)
	if err != nil {
		return nil, err
	}

	return &GardenRunner{
		SocketPath:    filepath.Join(tmpdir, "garden.sock"),
		DepotPath:     depotPath,
		RootPath:      rootPath,
		RootFSPath:    rootFSPath,
		SnapshotsPath: filepath.Join(tmpdir, "snapshots"),

		gardenBin: gardenBin,

		tmpdir: tmpdir,
	}, nil
}

func (r *GardenRunner) Start(argv ...string) error {
	garden := exec.Command(
		r.gardenBin,
		append(
			argv,
			"--socket", r.SocketPath,
			"--root", r.RootPath,
			"--depot", r.DepotPath,
			"--rootfs", r.RootFSPath,
			"--snapshots", r.SnapshotsPath,
			"--debug",
		)...,
	)

	garden.Stdout = os.Stdout
	garden.Stderr = os.Stderr

	err := garden.Start()
	if err != nil {
		return err
	}

	started := make(chan bool, 1)
	stop := make(chan bool, 1)

	go r.waitForStart(started, stop)

	timeout := 10 * time.Second

	r.gardenCmd = garden

	select {
	case <-started:
		return nil
	case <-time.After(timeout):
		stop <- true
		return fmt.Errorf("garden did not come up within %s", timeout)
	}
}

func (r *GardenRunner) Stop() error {
	if r.gardenCmd == nil {
		return nil
	}

	err := r.gardenCmd.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	stopped := make(chan bool, 1)
	stop := make(chan bool, 1)

	go r.waitForStop(stopped, stop)

	timeout := 10 * time.Second

	select {
	case <-stopped:
		r.gardenCmd = nil
		return nil
	case <-time.After(timeout):
		stop <- true
		return fmt.Errorf("garden did not shut down within %s", timeout)
	}
}

func (r *GardenRunner) DestroyContainers() error {
	containerDirs, err := ioutil.ReadDir(r.DepotPath)
	if err != nil {
		return err
	}

	for _, dir := range containerDirs {
		if dir.Name() == "tmp" {
			continue
		}

		destroy := exec.Command(
			filepath.Join(r.RootPath, "linux", "destroy.sh"),
			filepath.Join(r.DepotPath, dir.Name()),
		)

		err := destroy.Run()
		if err != nil {
			return err
		}
	}

	return os.RemoveAll(r.SnapshotsPath)
}

func (r *GardenRunner) TearDown() error {
	err := r.DestroyContainers()
	if err != nil {
		return err
	}

	return os.RemoveAll(r.tmpdir)
}

func (r *GardenRunner) waitForStart(started chan<- bool, stop <-chan bool) {
	for {
		conn, err := net.Dial("unix", r.SocketPath)
		if err == nil {
			conn.Close()
			started <- true
			return
		}

		select {
		case <-stop:
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (r *GardenRunner) waitForStop(stopped chan<- bool, stop <-chan bool) {
	for {
		conn, err := net.Dial("unix", r.SocketPath)
		if err != nil {
			stopped <- true
			return
		}

		conn.Close()

		select {
		case <-stop:
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}
