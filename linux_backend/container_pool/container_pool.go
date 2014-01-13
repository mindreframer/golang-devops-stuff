package container_pool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/command_runner"
	"github.com/vito/garden/linux_backend"
	"github.com/vito/garden/linux_backend/bandwidth_manager"
	"github.com/vito/garden/linux_backend/cgroups_manager"
	"github.com/vito/garden/linux_backend/network_pool"
	"github.com/vito/garden/linux_backend/quota_manager"
	"github.com/vito/garden/linux_backend/uid_pool"
)

type LinuxContainerPool struct {
	rootPath   string
	depotPath  string
	rootFSPath string

	uidPool     uid_pool.UIDPool
	networkPool network_pool.NetworkPool
	portPool    linux_backend.PortPool

	runner command_runner.CommandRunner

	quotaManager quota_manager.QuotaManager

	containerIDs chan string
}

func New(
	rootPath, depotPath, rootFSPath string,
	uidPool uid_pool.UIDPool,
	networkPool network_pool.NetworkPool,
	portPool linux_backend.PortPool,
	runner command_runner.CommandRunner,
	quotaManager quota_manager.QuotaManager,
) *LinuxContainerPool {
	pool := &LinuxContainerPool{
		rootPath:   rootPath,
		depotPath:  depotPath,
		rootFSPath: rootFSPath,

		uidPool:     uidPool,
		networkPool: networkPool,
		portPool:    portPool,

		runner: runner,

		quotaManager: quotaManager,

		containerIDs: make(chan string),
	}

	go pool.generateContainerIDs()

	return pool
}

func (p *LinuxContainerPool) Setup() error {
	setup := &exec.Cmd{
		Path: path.Join(p.rootPath, "setup.sh"),
		Env: []string{
			"POOL_NETWORK=" + p.networkPool.Network().String(),
			"ALLOW_NETWORKS=",
			"DENY_NETWORKS=",
			"CONTAINER_ROOTFS_PATH=" + p.rootFSPath,
			"CONTAINER_DEPOT_PATH=" + p.depotPath,
			"CONTAINER_DEPOT_MOUNT_POINT_PATH=" + p.quotaManager.MountPoint(),
			fmt.Sprintf("DISK_QUOTA_ENABLED=%v", p.quotaManager.IsEnabled()),

			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	err := p.runner.Run(setup)
	if err != nil {
		return err
	}

	return nil
}

func (p *LinuxContainerPool) Prune(keep map[string]bool) error {
	ls := &exec.Cmd{
		Path: "ls",
		Args: []string{p.depotPath},
	}

	out := new(bytes.Buffer)

	ls.Stdout = out

	err := p.runner.Run(ls)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(out)

	for {
		container, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		// trim linebreak
		id := container[0 : len(container)-1]

		if id == "tmp" {
			continue
		}

		_, found := keep[id]
		if found {
			continue
		}

		log.Println("pruning", id)

		err = p.destroy(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *LinuxContainerPool) Create(spec backend.ContainerSpec) (linux_backend.Container, error) {
	uid, err := p.uidPool.Acquire()
	if err != nil {
		return nil, err
	}

	network, err := p.networkPool.Acquire()
	if err != nil {
		p.uidPool.Release(uid)
		return nil, err
	}

	id := <-p.containerIDs

	containerPath := path.Join(p.depotPath, id)

	cgroupsManager := cgroups_manager.New("/tmp/warden/cgroup", id, p.runner)

	bandwidthManager := bandwidth_manager.New(containerPath, id, p.runner)

	handle := id
	if spec.Handle != "" {
		handle = spec.Handle
	}

	container := linux_backend.NewLinuxContainer(
		id,
		handle,
		containerPath,
		spec.GraceTime,
		linux_backend.NewResources(uid, network, []uint32{}),
		p.portPool,
		p.runner,
		cgroupsManager,
		p.quotaManager,
		bandwidthManager,
	)

	create := &exec.Cmd{
		Path: path.Join(p.rootPath, "create.sh"),
		Args: []string{containerPath},
		Env: []string{
			"id=" + container.ID(),
			"rootfs_path=" + p.rootFSPath,
			fmt.Sprintf("user_uid=%d", uid),
			fmt.Sprintf("network_host_ip=%s", network.HostIP()),
			fmt.Sprintf("network_container_ip=%s", network.ContainerIP()),
			"network_netmask=255.255.255.252",

			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	err = p.runner.Run(create)
	if err != nil {
		p.uidPool.Release(uid)
		p.networkPool.Release(network)
		return nil, err
	}

	err = p.writeBindMounts(containerPath, spec.BindMounts)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (p *LinuxContainerPool) Restore(snapshot io.Reader) (linux_backend.Container, error) {
	var containerSnapshot linux_backend.ContainerSnapshot

	err := json.NewDecoder(snapshot).Decode(&containerSnapshot)
	if err != nil {
		return nil, err
	}

	id := containerSnapshot.ID

	log.Println("restoring", id)

	resources := containerSnapshot.Resources

	err = p.uidPool.Remove(resources.UID)
	if err != nil {
		return nil, err
	}

	err = p.networkPool.Remove(resources.Network)
	if err != nil {
		p.uidPool.Release(resources.UID)
		return nil, err
	}

	for _, port := range resources.Ports {
		err = p.portPool.Remove(port)
		if err != nil {
			p.uidPool.Release(resources.UID)
			p.networkPool.Release(resources.Network)

			for _, port := range resources.Ports {
				p.portPool.Release(port)
			}

			return nil, err
		}
	}

	containerPath := path.Join(p.depotPath, id)

	cgroupsManager := cgroups_manager.New("/tmp/warden/cgroup", id, p.runner)

	bandwidthManager := bandwidth_manager.New(containerPath, id, p.runner)

	container := linux_backend.NewLinuxContainer(
		id,
		containerSnapshot.Handle,
		containerPath,
		containerSnapshot.GraceTime,
		linux_backend.NewResources(
			resources.UID,
			resources.Network,
			resources.Ports,
		),
		p.portPool,
		p.runner,
		cgroupsManager,
		p.quotaManager,
		bandwidthManager,
	)

	err = container.Restore(containerSnapshot)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (p *LinuxContainerPool) Destroy(container linux_backend.Container) error {
	err := p.destroy(container.ID())
	if err != nil {
		return err
	}

	linuxContainer := container.(*linux_backend.LinuxContainer)

	resources := linuxContainer.Resources()

	for _, port := range resources.Ports {
		p.portPool.Release(port)
	}

	p.uidPool.Release(resources.UID)

	p.networkPool.Release(resources.Network)

	return nil
}

func (p *LinuxContainerPool) destroy(id string) error {
	destroy := &exec.Cmd{
		Path: path.Join(p.rootPath, "destroy.sh"),
		Args: []string{path.Join(p.depotPath, id)},
	}

	return p.runner.Run(destroy)
}

func (p *LinuxContainerPool) generateContainerIDs() string {
	for containerNum := time.Now().UnixNano(); ; containerNum++ {
		containerID := []byte{}

		var i uint
		for i = 0; i < 11; i++ {
			containerID = strconv.AppendInt(
				containerID,
				(containerNum>>(55-(i+1)*5))&31,
				32,
			)
		}

		p.containerIDs <- string(containerID)
	}
}

func (p *LinuxContainerPool) writeBindMounts(
	containerPath string,
	bindMounts []backend.BindMount,
) error {
	hook := path.Join(containerPath, "lib", "hook-child-before-pivot.sh")

	for _, bm := range bindMounts {
		dstMount := path.Join(containerPath, "mnt", bm.DstPath)
		srcPath := path.Join(p.runner.ServerRoot(), bm.SrcPath)

		mode := "ro"
		if bm.Mode == backend.BindMountModeRW {
			mode = "rw"
		}

		linebreak := &exec.Cmd{
			Path: "bash",
			Args: []string{
				"-c",
				"echo >> " + hook,
			},
		}

		err := p.runner.Run(linebreak)
		if err != nil {
			return err
		}

		mkdir := &exec.Cmd{
			Path: "bash",
			Args: []string{
				"-c",
				"echo mkdir -p " + dstMount + " >> " + hook,
			},
		}

		err = p.runner.Run(mkdir)
		if err != nil {
			return err
		}

		mount := &exec.Cmd{
			Path: "bash",
			Args: []string{
				"-c",
				"echo mount -n --bind " + srcPath + " " + dstMount +
					" >> " + hook,
			},
		}

		err = p.runner.Run(mount)
		if err != nil {
			return err
		}

		remount := &exec.Cmd{
			Path: "bash",
			Args: []string{
				"-c",
				"echo mount -n --bind -o remount," + mode + " " + srcPath + " " + dstMount +
					" >> " + hook,
			},
		}

		err = p.runner.Run(remount)
		if err != nil {
			return err
		}
	}

	return nil
}
