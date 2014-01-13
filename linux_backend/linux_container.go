package linux_backend

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
	"strings"
	"sync"
	"time"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/command_runner"
	"github.com/vito/garden/linux_backend/bandwidth_manager"
	"github.com/vito/garden/linux_backend/cgroups_manager"
	"github.com/vito/garden/linux_backend/job_tracker"
	"github.com/vito/garden/linux_backend/quota_manager"
)

type LinuxContainer struct {
	id     string
	handle string
	path   string

	graceTime time.Duration

	state      State
	stateMutex sync.RWMutex

	events      []string
	eventsMutex sync.RWMutex

	resources *Resources

	portPool PortPool

	runner command_runner.CommandRunner

	cgroupsManager   cgroups_manager.CgroupsManager
	quotaManager     quota_manager.QuotaManager
	bandwidthManager bandwidth_manager.BandwidthManager

	jobTracker *job_tracker.JobTracker

	oomMutex    sync.RWMutex
	oomNotifier *exec.Cmd

	currentBandwidthLimits *backend.BandwidthLimits
	bandwidthMutex         sync.RWMutex

	currentDiskLimits *backend.DiskLimits
	diskMutex         sync.RWMutex

	currentMemoryLimits *backend.MemoryLimits
	memoryMutex         sync.RWMutex

	currentCPULimits *backend.CPULimits
	cpuMutex         sync.RWMutex

	netIns      []NetInSpec
	netInsMutex sync.RWMutex

	netOuts      []NetOutSpec
	netOutsMutex sync.RWMutex
}

type NetInSpec struct {
	HostPort      uint32
	ContainerPort uint32
}

type NetOutSpec struct {
	Network string
	Port    uint32
}

type PortPool interface {
	Acquire() (uint32, error)
	Remove(uint32) error
	Release(uint32)
}

type State string

const (
	StateBorn    = State("born")
	StateActive  = State("active")
	StateStopped = State("stopped")
)

func NewLinuxContainer(
	id, handle, path string,
	graceTime time.Duration,
	resources *Resources,
	portPool PortPool,
	runner command_runner.CommandRunner,
	cgroupsManager cgroups_manager.CgroupsManager,
	quotaManager quota_manager.QuotaManager,
	bandwidthManager bandwidth_manager.BandwidthManager,
) *LinuxContainer {
	return &LinuxContainer{
		id:     id,
		handle: handle,
		path:   path,

		graceTime: graceTime,

		state:  StateBorn,
		events: []string{},

		resources: resources,

		portPool: portPool,

		runner: runner,

		cgroupsManager:   cgroupsManager,
		quotaManager:     quotaManager,
		bandwidthManager: bandwidthManager,

		jobTracker: job_tracker.New(path, runner),
	}
}

func (c *LinuxContainer) ID() string {
	return c.id
}

func (c *LinuxContainer) Handle() string {
	return c.handle
}

func (c *LinuxContainer) GraceTime() time.Duration {
	return c.graceTime
}

func (c *LinuxContainer) State() State {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	return c.state
}

func (c *LinuxContainer) Events() []string {
	c.eventsMutex.RLock()
	defer c.eventsMutex.RUnlock()

	events := make([]string, len(c.events))

	copy(events, c.events)

	return events
}

func (c *LinuxContainer) Resources() *Resources {
	return c.resources
}

func (c *LinuxContainer) Snapshot(out io.Writer) error {
	c.bandwidthMutex.RLock()
	defer c.bandwidthMutex.RUnlock()

	c.cpuMutex.RLock()
	defer c.cpuMutex.RUnlock()

	c.diskMutex.RLock()
	defer c.diskMutex.RUnlock()

	c.memoryMutex.RLock()
	defer c.memoryMutex.RUnlock()

	c.netInsMutex.RLock()
	defer c.netInsMutex.RUnlock()

	c.netOutsMutex.RLock()
	defer c.netOutsMutex.RUnlock()

	jobSnapshots := []JobSnapshot{}

	for _, job := range c.jobTracker.ActiveJobs() {
		jobSnapshots = append(
			jobSnapshots,
			JobSnapshot{ID: job.ID, DiscardOutput: job.DiscardOutput},
		)
	}

	return json.NewEncoder(out).Encode(
		ContainerSnapshot{
			ID:     c.id,
			Handle: c.handle,

			GraceTime: c.graceTime,

			State:  string(c.State()),
			Events: c.Events(),

			Limits: LimitsSnapshot{
				Bandwidth: c.currentBandwidthLimits,
				CPU:       c.currentCPULimits,
				Disk:      c.currentDiskLimits,
				Memory:    c.currentMemoryLimits,
			},

			Resources: ResourcesSnapshot{
				UID:     c.resources.UID,
				Network: c.resources.Network,
				Ports:   c.resources.Ports,
			},

			NetIns:  c.netIns,
			NetOuts: c.netOuts,

			Jobs: jobSnapshots,
		},
	)
}

func (c *LinuxContainer) Restore(snapshot ContainerSnapshot) error {
	c.setState(State(snapshot.State))

	for _, ev := range snapshot.Events {
		c.registerEvent(ev)
	}

	if snapshot.Limits.Memory != nil {
		err := c.LimitMemory(*snapshot.Limits.Memory)
		if err != nil {
			return err
		}
	}

	for _, job := range snapshot.Jobs {
		c.jobTracker.Restore(job.ID, job.DiscardOutput)
	}

	net := &exec.Cmd{
		Path: path.Join(c.path, "net.sh"),
		Args: []string{"setup"},
	}

	err := c.runner.Run(net)
	if err != nil {
		return err
	}

	for _, in := range snapshot.NetIns {
		_, _, err = c.NetIn(in.HostPort, in.ContainerPort)
		if err != nil {
			return err
		}
	}

	for _, out := range snapshot.NetOuts {
		err = c.NetOut(out.Network, out.Port)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *LinuxContainer) Start() error {
	log.Println(c.id, "starting")

	start := &exec.Cmd{
		Path: path.Join(c.path, "start.sh"),
		Env: []string{
			"id=" + c.id,
			"container_iface_mtu=1500",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	err := c.runner.Run(start)
	if err != nil {
		return err
	}

	c.setState(StateActive)

	return nil
}

func (c *LinuxContainer) Stop(kill bool) error {
	log.Println(c.id, "stopping")

	stop := &exec.Cmd{
		Path: path.Join(c.path, "stop.sh"),
	}

	if kill {
		stop.Args = append(stop.Args, "-w", "0")
	}

	err := c.runner.Run(stop)
	if err != nil {
		return err
	}

	c.stopOomNotifier()

	c.setState(StateStopped)

	return nil
}

func (c *LinuxContainer) Cleanup() {
	c.stopOomNotifier()

	for _, job := range c.jobTracker.ActiveJobs() {
		job.Unlink()
	}
}

func (c *LinuxContainer) Info() (backend.ContainerInfo, error) {
	log.Println(c.id, "info")

	memoryStat, err := c.cgroupsManager.Get("memory", "memory.stat")
	if err != nil {
		return backend.ContainerInfo{}, err
	}

	cpuUsage, err := c.cgroupsManager.Get("cpuacct", "cpuacct.usage")
	if err != nil {
		return backend.ContainerInfo{}, err
	}

	cpuStat, err := c.cgroupsManager.Get("cpuacct", "cpuacct.stat")
	if err != nil {
		return backend.ContainerInfo{}, err
	}

	diskStat, err := c.quotaManager.GetUsage(c.resources.UID)
	if err != nil {
		return backend.ContainerInfo{}, err
	}

	bandwidthStat, err := c.bandwidthManager.GetLimits()
	if err != nil {
		return backend.ContainerInfo{}, err
	}

	jobIDs := []uint32{}
	for _, job := range c.jobTracker.ActiveJobs() {
		jobIDs = append(jobIDs, job.ID)
	}

	return backend.ContainerInfo{
		State:         string(c.State()),
		Events:        c.Events(),
		HostIP:        c.resources.Network.HostIP().String(),
		ContainerIP:   c.resources.Network.ContainerIP().String(),
		ContainerPath: c.path,
		JobIDs:        jobIDs,
		MemoryStat:    parseMemoryStat(memoryStat),
		CPUStat:       parseCPUStat(cpuUsage, cpuStat),
		DiskStat:      diskStat,
		BandwidthStat: bandwidthStat,
	}, nil
}

func (c *LinuxContainer) CopyIn(src, dst string) error {
	log.Println(c.id, "copying in from", src, "to", dst)
	return c.rsync(path.Join(c.runner.ServerRoot(), src), "vcap@container:"+dst)
}

func (c *LinuxContainer) CopyOut(src, dst, owner string) error {
	log.Println(c.id, "copying out from", src, "to", dst)

	dstDir := path.Join(c.runner.ServerRoot(), dst)

	err := c.rsync("vcap@container:"+src, dstDir)
	if err != nil {
		return err
	}

	if owner != "" {
		chown := &exec.Cmd{
			Path: "chown",
			Args: []string{"-R", owner, dstDir},
		}

		err := c.runner.Run(chown)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *LinuxContainer) LimitBandwidth(limits backend.BandwidthLimits) error {
	log.Println(
		c.id,
		"limiting bandwidth to",
		limits.RateInBytesPerSecond,
		"bytes per second; burst",
		limits.BurstRateInBytesPerSecond,
	)

	err := c.bandwidthManager.SetLimits(limits)
	if err != nil {
		return err
	}

	c.bandwidthMutex.Lock()
	defer c.bandwidthMutex.Unlock()

	c.currentBandwidthLimits = &limits

	return nil
}

func (c *LinuxContainer) CurrentBandwidthLimits() (backend.BandwidthLimits, error) {
	c.bandwidthMutex.RLock()
	defer c.bandwidthMutex.RUnlock()

	if c.currentBandwidthLimits == nil {
		return backend.BandwidthLimits{}, nil
	}

	return *c.currentBandwidthLimits, nil
}

func (c *LinuxContainer) LimitDisk(limits backend.DiskLimits) error {
	err := c.quotaManager.SetLimits(c.resources.UID, limits)
	if err != nil {
		return err
	}

	c.diskMutex.Lock()
	defer c.diskMutex.Unlock()

	c.currentDiskLimits = &limits

	return nil
}

func (c *LinuxContainer) CurrentDiskLimits() (backend.DiskLimits, error) {
	return c.quotaManager.GetLimits(c.resources.UID)
}

func (c *LinuxContainer) LimitMemory(limits backend.MemoryLimits) error {
	log.Println(c.id, "limiting memory to", limits.LimitInBytes, "bytes")

	err := c.startOomNotifier()
	if err != nil {
		return err
	}

	limit := fmt.Sprintf("%d", limits.LimitInBytes)

	// memory.memsw.limit_in_bytes must be >= memory.limit_in_bytes
	//
	// however, it must be set after memory.limit_in_bytes, and if we're
	// increasing the limit, writing memory.limit_in_bytes first will fail.
	//
	// so, write memory.limit_in_bytes before and after
	c.cgroupsManager.Set("memory", "memory.limit_in_bytes", limit)

	err = c.cgroupsManager.Set("memory", "memory.memsw.limit_in_bytes", limit)
	if err != nil {
		return err
	}

	err = c.cgroupsManager.Set("memory", "memory.limit_in_bytes", limit)
	if err != nil {
		return err
	}

	c.memoryMutex.Lock()
	defer c.memoryMutex.Unlock()

	c.currentMemoryLimits = &limits

	return nil
}

func (c *LinuxContainer) CurrentMemoryLimits() (backend.MemoryLimits, error) {
	limitInBytes, err := c.cgroupsManager.Get("memory", "memory.limit_in_bytes")
	if err != nil {
		return backend.MemoryLimits{}, err
	}

	numericLimit, err := strconv.ParseUint(limitInBytes, 10, 0)
	if err != nil {
		return backend.MemoryLimits{}, err
	}

	return backend.MemoryLimits{uint64(numericLimit)}, nil
}

func (c *LinuxContainer) LimitCPU(limits backend.CPULimits) error {
	log.Println(c.id, "limiting CPU to", limits.LimitInShares, "shares")

	limit := fmt.Sprintf("%d", limits.LimitInShares)

	err := c.cgroupsManager.Set("cpu", "cpu.shares", limit)
	if err != nil {
		return err
	}

	c.cpuMutex.Lock()
	defer c.cpuMutex.Unlock()

	c.currentCPULimits = &limits

	return nil
}

func (c *LinuxContainer) CurrentCPULimits() (backend.CPULimits, error) {
	actualLimitInShares, err := c.cgroupsManager.Get("cpu", "cpu.shares")
	if err != nil {
		return backend.CPULimits{}, err
	}

	numericLimit, err := strconv.ParseUint(actualLimitInShares, 10, 0)
	if err != nil {
		return backend.CPULimits{}, err
	}

	return backend.CPULimits{uint64(numericLimit)}, nil
}

func (c *LinuxContainer) Spawn(spec backend.JobSpec) (uint32, error) {
	log.Println(c.id, "spawning job:", spec.Script)

	wshPath := path.Join(c.path, "bin", "wsh")
	sockPath := path.Join(c.path, "run", "wshd.sock")

	user := "vcap"
	if spec.Privileged {
		user = "root"
	}

	wsh := &exec.Cmd{
		Path:  wshPath,
		Args:  []string{"--socket", sockPath, "--user", user, "/bin/bash"},
		Stdin: bytes.NewBufferString(spec.Script),
	}

	setRLimitsEnv(wsh, spec.Limits)

	return c.jobTracker.Spawn(wsh, spec.DiscardOutput, spec.AutoLink)
}

func (c *LinuxContainer) Stream(jobID uint32) (<-chan backend.JobStream, error) {
	log.Println(c.id, "streaming job", jobID)
	return c.jobTracker.Stream(jobID)
}

func (c *LinuxContainer) Link(jobID uint32) (backend.JobResult, error) {
	log.Println(c.id, "linking to job", jobID)

	exitStatus, stdout, stderr, err := c.jobTracker.Link(jobID)
	if err != nil {
		return backend.JobResult{}, err
	}

	return backend.JobResult{
		ExitStatus: exitStatus,
		Stdout:     stdout,
		Stderr:     stderr,
	}, nil
}

func (c *LinuxContainer) NetIn(hostPort uint32, containerPort uint32) (uint32, uint32, error) {
	if hostPort == 0 {
		randomPort, err := c.portPool.Acquire()
		if err != nil {
			return 0, 0, err
		}

		c.resources.AddPort(randomPort)

		hostPort = randomPort
	}

	if containerPort == 0 {
		containerPort = hostPort
	}

	log.Println(
		c.id,
		"mapping host port",
		hostPort,
		"to container port",
		containerPort,
	)

	net := &exec.Cmd{
		Path: path.Join(c.path, "net.sh"),
		Args: []string{"in"},
		Env: []string{
			fmt.Sprintf("HOST_PORT=%d", hostPort),
			fmt.Sprintf("CONTAINER_PORT=%d", containerPort),
		},
	}

	err := c.runner.Run(net)
	if err != nil {
		return 0, 0, err
	}

	c.netInsMutex.Lock()
	defer c.netInsMutex.Unlock()

	c.netIns = append(c.netIns, NetInSpec{hostPort, containerPort})

	return hostPort, containerPort, nil
}

func (c *LinuxContainer) NetOut(network string, port uint32) error {
	net := &exec.Cmd{
		Path: path.Join(c.path, "net.sh"),
		Args: []string{"out"},
	}

	if port != 0 {
		log.Println(
			c.id,
			"permitting traffic to",
			network,
			"with port",
			port,
		)

		net.Env = []string{
			"NETWORK=" + network,
			fmt.Sprintf("PORT=%d", port),
		}
	} else {
		if network == "" {
			return fmt.Errorf("network and/or port must be provided")
		}

		log.Println(c.id, "permitting traffic to", network)

		net.Env = []string{
			"NETWORK=" + network,
			"PORT=",
		}
	}

	err := c.runner.Run(net)
	if err != nil {
		return err
	}

	c.netOutsMutex.Lock()
	defer c.netOutsMutex.Unlock()

	c.netOuts = append(c.netOuts, NetOutSpec{network, port})

	return nil
}

func (c *LinuxContainer) setState(state State) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	c.state = state
}

func (c *LinuxContainer) registerEvent(event string) {
	c.eventsMutex.Lock()
	defer c.eventsMutex.Unlock()

	c.events = append(c.events, event)
}

func (c *LinuxContainer) rsync(src, dst string) error {
	wshPath := path.Join(c.path, "bin", "wsh")
	sockPath := path.Join(c.path, "run", "wshd.sock")

	rsync := &exec.Cmd{
		Path: "rsync",
		Args: []string{
			"-e", wshPath + " --socket " + sockPath + " --rsh",
			"-r",
			"-p",
			"--links",
			src,
			dst,
		},
	}

	return c.runner.Run(rsync)
}

func (c *LinuxContainer) startOomNotifier() error {
	c.oomMutex.Lock()
	defer c.oomMutex.Unlock()

	if c.oomNotifier != nil {
		return nil
	}

	oomPath := path.Join(c.path, "bin", "oom")

	c.oomNotifier = &exec.Cmd{
		Path: oomPath,
		Args: []string{c.cgroupsManager.SubsystemPath("memory")},
	}

	err := c.runner.Start(c.oomNotifier)
	if err != nil {
		return err
	}

	go c.watchForOom(c.oomNotifier)

	return nil
}

func (c *LinuxContainer) stopOomNotifier() {
	c.oomMutex.RLock()
	defer c.oomMutex.RUnlock()

	if c.oomNotifier != nil {
		c.runner.Kill(c.oomNotifier)
	}
}

func (c *LinuxContainer) watchForOom(oom *exec.Cmd) {
	err := c.runner.Wait(oom)
	if err == nil {
		log.Println(c.id, "out of memory")
		c.registerEvent("out of memory")
		c.Stop(false)
	} else {
		log.Println(c.id, "oom failed:", err)
	}

	// TODO: handle case where oom notifier itself failed? kill container?
}

func parseMemoryStat(contents string) (stat backend.ContainerMemoryStat) {
	scanner := bufio.NewScanner(strings.NewReader(contents))

	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		field := scanner.Text()

		if !scanner.Scan() {
			break
		}

		value, err := strconv.ParseUint(scanner.Text(), 10, 0)
		if err != nil {
			continue
		}

		switch field {
		case "cache":
			stat.Cache = value
		case "rss":
			stat.Rss = value
		case "mapped_file":
			stat.MappedFile = value
		case "pgpgin":
			stat.Pgpgin = value
		case "pgpgout":
			stat.Pgpgout = value
		case "swap":
			stat.Swap = value
		case "pgfault":
			stat.Pgfault = value
		case "pgmajfault":
			stat.Pgmajfault = value
		case "inactive_anon":
			stat.InactiveAnon = value
		case "active_anon":
			stat.ActiveAnon = value
		case "inactive_file":
			stat.InactiveFile = value
		case "active_file":
			stat.ActiveFile = value
		case "unevictable":
			stat.Unevictable = value
		case "hierarchical_memory_limit":
			stat.HierarchicalMemoryLimit = value
		case "hierarchical_memsw_limit":
			stat.HierarchicalMemswLimit = value
		case "total_cache":
			stat.TotalCache = value
		case "total_rss":
			stat.TotalRss = value
		case "total_mapped_file":
			stat.TotalMappedFile = value
		case "total_pgpgin":
			stat.TotalPgpgin = value
		case "total_pgpgout":
			stat.TotalPgpgout = value
		case "total_swap":
			stat.TotalSwap = value
		case "total_pgfault":
			stat.TotalPgfault = value
		case "total_pgmajfault":
			stat.TotalPgmajfault = value
		case "total_inactive_anon":
			stat.TotalInactiveAnon = value
		case "total_active_anon":
			stat.TotalActiveAnon = value
		case "total_inactive_file":
			stat.TotalInactiveFile = value
		case "total_active_file":
			stat.TotalActiveFile = value
		case "total_unevictable":
			stat.TotalUnevictable = value
		}
	}

	return
}

func parseCPUStat(usage, statContents string) (stat backend.ContainerCPUStat) {
	cpuUsage, err := strconv.ParseUint(strings.Trim(usage, "\n"), 10, 0)
	if err != nil {
		return
	}

	stat.Usage = cpuUsage

	scanner := bufio.NewScanner(strings.NewReader(statContents))

	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		field := scanner.Text()

		if !scanner.Scan() {
			break
		}

		value, err := strconv.ParseUint(scanner.Text(), 10, 0)
		if err != nil {
			continue
		}

		switch field {
		case "user":
			stat.User = value
		case "system":
			stat.System = value
		}
	}

	return
}

func setRLimitsEnv(cmd *exec.Cmd, rlimits backend.ResourceLimits) {
	if rlimits.As != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_AS=%d", *rlimits.As))
	}

	if rlimits.Core != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_CORE=%d", *rlimits.Core))
	}

	if rlimits.Cpu != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_CPU=%d", *rlimits.Cpu))
	}

	if rlimits.Data != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_DATA=%d", *rlimits.Data))
	}

	if rlimits.Fsize != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_FSIZE=%d", *rlimits.Fsize))
	}

	if rlimits.Locks != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_LOCKS=%d", *rlimits.Locks))
	}

	if rlimits.Memlock != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_MEMLOCK=%d", *rlimits.Memlock))
	}

	if rlimits.Msgqueue != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_MSGQUEUE=%d", *rlimits.Msgqueue))
	}

	if rlimits.Nice != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_NICE=%d", *rlimits.Nice))
	}

	if rlimits.Nofile != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_NOFILE=%d", *rlimits.Nofile))
	}

	if rlimits.Nproc != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_NPROC=%d", *rlimits.Nproc))
	}

	if rlimits.Rss != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_RSS=%d", *rlimits.Rss))
	}

	if rlimits.Rtprio != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_RTPRIO=%d", *rlimits.Rtprio))
	}

	if rlimits.Sigpending != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_SIGPENDING=%d", *rlimits.Sigpending))
	}

	if rlimits.Stack != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RLIMIT_STACK=%d", *rlimits.Stack))
	}
}
