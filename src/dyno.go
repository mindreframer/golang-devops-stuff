package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type (
	Dyno struct {
		Host, Container, Application, Process, Version, Port string
	}
	NodeStatusRunning struct {
		status  NodeStatus
		running bool
	}
	NodeStatuses  []NodeStatusRunning
	DynoGenerator struct {
		server      *Server
		statuses    []NodeStatusRunning
		position    int
		application string
		version     string
		usedPorts   []int
	}
	DynoPortTracker struct {
		allocations map[string][]int
		lock        sync.Mutex
	}
)

const (
	DYNO_DELIMITER = "_"
)

var (
	dynoPortTracker = DynoPortTracker{allocations: map[string][]int{}, lock: sync.Mutex{}}
)

// Check if a port is already in use.
func (this *DynoPortTracker) AlreadyInUse(host string, port int) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if ports, ok := this.allocations[host]; ok {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
	}
	return false
}

// Attempt to allocate a port for a node host.
func (this *DynoPortTracker) Allocate(host string, port int) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if ports, ok := this.allocations[host]; ok {
		// Require that the port not be already in use.
		for _, p := range ports {
			if p == port {
				return fmt.Errorf("Host/port combination %v/%v is already in use", host, port)
			}
		}
		this.allocations[host] = append(ports, port)
	} else {
		this.allocations[host] = []int{port}
	}
	// Schedule the port to be automatically freed once the status monitor will have picked up the in-use port.
	go func(host string, port int) {
		time.Sleep(300 * time.Second)
		this.Release(host, port)
	}(host, port)
	fmt.Printf("DynoPortTracker.Allocate :: added host=%v port=%v\n", host, port)
	return nil
}

// Release a previously allocated host/port pair if it is still in the allocations table.
func (this *DynoPortTracker) Release(host string, port int) {
	fmt.Printf("DynoPortTracker.Release :: removing host=%v port=%v\n", host, port)
	this.lock.Lock()
	defer this.lock.Unlock()
	if ports, ok := this.allocations[host]; ok {
		newPorts := []int{}
		for _, p := range ports {
			if p != port {
				newPorts = append(newPorts, p)
			}
		}
		this.allocations[host] = newPorts
	}
}

// NB: Container name format is: appName-version-process-port
func ContainerToDyno(host string, container string) (Dyno, error) {
	tokens := strings.Split(container, DYNO_DELIMITER)
	if len(tokens) != 4 {
		return Dyno{}, fmt.Errorf("Failed to parse container string '%v' into 4 tokens", container)
	}
	return Dyno{
		Host:        host,
		Container:   container,
		Application: tokens[0],
		Version:     tokens[1],
		Process:     tokens[2],
		Port:        tokens[3],
	}, nil
}

func NodeStatusToDynos(nodeStatus *NodeStatus) ([]Dyno, error) {
	dynos := make([]Dyno, len(nodeStatus.Containers))
	for i, container := range nodeStatus.Containers {
		dyno, err := ContainerToDyno(nodeStatus.Host, container)
		if err != nil {
			return dynos, err
		}
		dynos[i] = dyno
	}
	return dynos, nil
}

func (this *Dyno) Shutdown(e *Executor) error {
	fmt.Fprintf(e.logger, "Shutting down dyno, host=%v app=%v version=%v proc=%v port=%v", this.Host, this.Application, this.Version, this.Process, this.Port)
	return e.Run("ssh", DEFAULT_NODE_USERNAME+"@"+this.Host, "sudo", "/tmp/shutdown_container.py", this.Container)
}

func (this *Server) GetRunningDynos(application, process string) ([]Dyno, error) {
	dynos := []Dyno{}

	cfg, err := this.getConfig(true)
	if err != nil {
		return dynos, err
	}

	for _, node := range cfg.Nodes {
		status := this.getNodeStatus(node)
		// skip this node if there's an error
		if status.Err != nil {
			continue
		}
		for _, container := range status.Containers {
			dyno, err := ContainerToDyno(node.Host, container)
			if err != nil {
				fmt.Printf("Container->Dyno parse failed: %v", err)
			} else if dyno.Application == application && dyno.Process == process {
				dynos = append(dynos, dyno)
			}
		}
	}
	return dynos, nil
}

/* NB: THIS IS DEPRECATED, THE `done` CHANNEL IS ERROR PRONE AND WEIRD.
func (this *Server) selectNextDynos(nodes []*Node, application, process string, version string, done chan bool) (chan Dyno, error) {
	resultChannel := make(chan Dyno)
	// Produce sorted sequence of NodeStatuses.
	allStatuses := []NodeStatusRunning{}
	for _, node := range nodes {
		running := false
		nodeStatus := this.getNodeStatus(node)
		// Determine if there is an identical app/version container already running on the node.
		for _, container := range nodeStatus.Containers {
			dyno, _ := ContainerToDyno(node.Host, container)
			if dyno.Application == application && dyno.Version == version {
				running = true
				break
			}
		}
		allStatuses = append(allStatuses, NodeStatusRunning{nodeStatus, running})
	}

	if len(allStatuses) == 0 {
		return nil, fmt.Errorf("The node list was empty, which means deployment is impossible")
	}

	sort.Sort(NodeStatuses(allStatuses))

	go func() {
	OUTER:
		for i := 0; true; i++ {
			nodeStatus := allStatuses[i%len(allStatuses)].status
			port := fmt.Sprint(this.getNextPort(&nodeStatus))
			dyno, _ := ContainerToDyno(nodeStatus.Host, application+DYNO_DELIMITER+version+DYNO_DELIMITER+process+DYNO_DELIMITER+port)
			select {
			case <-done:
				break OUTER // Exit the infinite for-loop.
			case resultChannel <- dyno: // Send dyno to resultChannel.
			}
		}
	}()

	return resultChannel, nil
}*/

// Decicde which nodes to run the next N-count dynos on.
func (this *Server) NewDynoGenerator(nodes []*Node, application string, version string) (*DynoGenerator, error) {
	// Produce sorted sequence of NodeStatuses.
	allStatuses := []NodeStatusRunning{}
	for _, node := range nodes {
		running := false
		nodeStatus := this.getNodeStatus(node)
		// Determine if there is an identical app/version container already running on the node.
		for _, container := range nodeStatus.Containers {
			dyno, _ := ContainerToDyno(node.Host, container)
			if dyno.Application == application && dyno.Version == version {
				running = true
				break
			}
		}
		allStatuses = append(allStatuses, NodeStatusRunning{nodeStatus, running})
	}

	if len(allStatuses) == 0 {
		return nil, fmt.Errorf("The node list was empty, which means deployment is impossible")
	}

	sort.Sort(NodeStatuses(allStatuses))

	return &DynoGenerator{
		server:      this,
		statuses:    allStatuses,
		position:    0,
		application: application,
		version:     version,
		usedPorts:   []int{},
	}, nil

}

func (this *DynoGenerator) Next(process string) Dyno {
	nodeStatus := this.statuses[this.position%len(this.statuses)].status
	this.position++
	port := fmt.Sprint(this.server.getNextPort(&nodeStatus, &this.usedPorts))
	dyno, _ := ContainerToDyno(nodeStatus.Host, this.application+DYNO_DELIMITER+this.version+DYNO_DELIMITER+process+DYNO_DELIMITER+port)
	return dyno
}

// NodeStatus sorting.
func (this NodeStatuses) Len() int { return len(this) } // boilerplate.

// NodeStatus sorting.
func (this NodeStatuses) Swap(i int, j int) { this[i], this[j] = this[j], this[i] } // boilerplate.

// NodeStatus sorting.
func (this NodeStatuses) Less(i int, j int) bool { // actual sorting logic.
	if this[i].running && !this[j].running {
		return true
	}
	if !this[i].running && this[j].running {
		return false
	}
	return this[i].status.FreeMemoryMb > this[j].status.FreeMemoryMb
}

func AppendIfMissing(slice []int, i int) []int {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// Get the next available port for a node.
func (this *Server) getNextPort(nodeStatus *NodeStatus, usedPorts *[]int) int {
	port := 10000
	for _, container := range nodeStatus.Containers {
		foundPort, _ := strconv.Atoi(container[strings.LastIndex(container, DYNO_DELIMITER)+1:])
		if foundPort > 0 {
			*usedPorts = AppendIfMissing(*usedPorts, foundPort)
		}
	}
	sort.Ints(*usedPorts)
	fmt.Printf("Server.getNextPort :: Found used ports: %v\n", *usedPorts)
	for _, usedPort := range *usedPorts {
		if port == usedPort || dynoPortTracker.AlreadyInUse(nodeStatus.Host, port) {
			port++
		} else if usedPort > port {
			break
		}
	}
	err := dynoPortTracker.Allocate(nodeStatus.Host, port)
	if err != nil {
		fmt.Printf("Server.getNextPort :: host/port combination %v/%v already in use, will find another\n", nodeStatus.Host, port)
		*usedPorts = AppendIfMissing(*usedPorts, port)
		return this.getNextPort(nodeStatus, usedPorts)
	}
	fmt.Printf("Server.getNextPort :: Result port: %v\n", port)
	*usedPorts = AppendIfMissing(*usedPorts, port)
	return port
}
