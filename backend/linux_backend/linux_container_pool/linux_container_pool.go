package linux_container_pool

import (
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/linux_backend"
	"github.com/vito/garden/backend/linux_backend/bandwidth_manager"
	"github.com/vito/garden/backend/linux_backend/cgroups_manager"
	"github.com/vito/garden/backend/linux_backend/network_pool"
	"github.com/vito/garden/backend/linux_backend/quota_manager"
	"github.com/vito/garden/backend/linux_backend/uid_pool"
	"github.com/vito/garden/command_runner"
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

	nextContainer int64

	sync.RWMutex
}

func New(
	rootPath, depotPath, rootFSPath string,
	uidPool uid_pool.UIDPool,
	networkPool network_pool.NetworkPool,
	portPool linux_backend.PortPool,
	runner command_runner.CommandRunner,
	quotaManager quota_manager.QuotaManager,
) *LinuxContainerPool {
	return &LinuxContainerPool{
		rootPath:   rootPath,
		depotPath:  depotPath,
		rootFSPath: rootFSPath,

		uidPool:     uidPool,
		networkPool: networkPool,
		portPool:    portPool,

		runner: runner,

		quotaManager: quotaManager,

		nextContainer: time.Now().UnixNano(),
	}
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

func (p *LinuxContainerPool) Create(spec backend.ContainerSpec) (backend.Container, error) {
	uid, err := p.uidPool.Acquire()
	if err != nil {
		return nil, err
	}

	network, err := p.networkPool.Acquire()
	if err != nil {
		p.uidPool.Release(uid)
		return nil, err
	}

	p.Lock()

	id := p.generateContainerID()

	p.Unlock()

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
		&linux_backend.Resources{
			UID:     uid,
			Network: network,
		},
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

func (p *LinuxContainerPool) Destroy(container backend.Container) error {
	destroy := &exec.Cmd{
		Path: path.Join(p.rootPath, "destroy.sh"),
		Args: []string{path.Join(p.depotPath, container.ID())},
		Env: []string{
			"id=" + container.ID(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	err := p.runner.Run(destroy)
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

func (p *LinuxContainerPool) generateContainerID() string {
	p.nextContainer++

	containerID := []byte{}

	var i uint
	for i = 0; i < 11; i++ {
		containerID = strconv.AppendInt(
			containerID,
			(p.nextContainer>>(55-(i+1)*5))&31,
			32,
		)
	}

	return string(containerID)
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
