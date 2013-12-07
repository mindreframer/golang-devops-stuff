package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"code.google.com/p/gogoprotobuf/proto"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/message_reader"
	protocol "github.com/vito/garden/protocol"
)

type WardenServer struct {
	socketPath string
	backend    backend.Backend
}

type UnhandledRequestError struct {
	Request proto.Message
}

func (e UnhandledRequestError) Error() string {
	return fmt.Sprintf("unhandled request type: %T", e.Request)
}

func New(socketPath string, backend backend.Backend) *WardenServer {
	return &WardenServer{
		socketPath: socketPath,
		backend:    backend,
	}
}

func (s *WardenServer) Start() error {
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}

	os.Chmod(s.socketPath, 0777)

	go s.handleConnections(listener)

	return nil
}

func (s *WardenServer) handleConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
			continue
		}

		go s.serveConnection(conn)
	}
}

func (s *WardenServer) serveConnection(conn net.Conn) {
	read := bufio.NewReader(conn)

	for {
		var response proto.Message
		var err error

		request, err := message_reader.ReadRequest(read)
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("error reading request:", err)
			continue
		}

		switch request.(type) {
		case *protocol.PingRequest:
			response, err = s.handlePing(request.(*protocol.PingRequest))
		case *protocol.EchoRequest:
			response, err = s.handleEcho(request.(*protocol.EchoRequest))
		case *protocol.CreateRequest:
			response, err = s.handleCreate(request.(*protocol.CreateRequest))
		case *protocol.DestroyRequest:
			response, err = s.handleDestroy(request.(*protocol.DestroyRequest))
		case *protocol.ListRequest:
			response, err = s.handleList(request.(*protocol.ListRequest))
		case *protocol.StopRequest:
			response, err = s.handleStop(request.(*protocol.StopRequest))
		case *protocol.CopyInRequest:
			response, err = s.handleCopyIn(request.(*protocol.CopyInRequest))
		case *protocol.CopyOutRequest:
			response, err = s.handleCopyOut(request.(*protocol.CopyOutRequest))
		case *protocol.SpawnRequest:
			response, err = s.handleSpawn(request.(*protocol.SpawnRequest))
		case *protocol.LinkRequest:
			response, err = s.handleLink(request.(*protocol.LinkRequest))
		case *protocol.StreamRequest:
			response, err = s.handleStream(conn, request.(*protocol.StreamRequest))
		case *protocol.RunRequest:
			response, err = s.handleRun(request.(*protocol.RunRequest))
		case *protocol.LimitBandwidthRequest:
			response, err = s.handleLimitBandwidth(request.(*protocol.LimitBandwidthRequest))
		case *protocol.LimitMemoryRequest:
			response, err = s.handleLimitMemory(request.(*protocol.LimitMemoryRequest))
		case *protocol.LimitDiskRequest:
			response, err = s.handleLimitDisk(request.(*protocol.LimitDiskRequest))
		case *protocol.NetInRequest:
			response, err = s.handleNetIn(request.(*protocol.NetInRequest))
		case *protocol.NetOutRequest:
			response, err = s.handleNetOut(request.(*protocol.NetOutRequest))
		case *protocol.InfoRequest:
			response, err = s.handleInfo(request.(*protocol.InfoRequest))
		default:
			err = UnhandledRequestError{request}
		}

		if err != nil {
			response = &protocol.ErrorResponse{
				Message: proto.String(err.Error()),
			}
		}

		protocol.Messages(response).WriteTo(conn)
	}
}

func (s *WardenServer) handlePing(ping *protocol.PingRequest) (proto.Message, error) {
	return &protocol.PingResponse{}, nil
}

func (s *WardenServer) handleEcho(echo *protocol.EchoRequest) (proto.Message, error) {
	return &protocol.EchoResponse{Message: echo.Message}, nil
}

func (s *WardenServer) handleCreate(create *protocol.CreateRequest) (proto.Message, error) {
	bindMounts := []backend.BindMount{}

	for _, bm := range create.GetBindMounts() {
		bindMount := backend.BindMount{
			SrcPath: bm.GetSrcPath(),
			DstPath: bm.GetDstPath(),
			Mode:    backend.BindMountMode(bm.GetMode()),
		}

		bindMounts = append(bindMounts, bindMount)
	}

	container, err := s.backend.Create(backend.ContainerSpec{
		Handle:     create.GetHandle(),
		GraceTime:  time.Duration(create.GetGraceTime()) * time.Second,
		RootFSPath: create.GetRootfs(),
		Network:    create.GetNetwork(),
		BindMounts: bindMounts,
	})

	if err != nil {
		return nil, err
	}

	return &protocol.CreateResponse{
		Handle: proto.String(container.Handle()),
	}, nil
}

func (s *WardenServer) handleDestroy(destroy *protocol.DestroyRequest) (proto.Message, error) {
	handle := destroy.GetHandle()

	err := s.backend.Destroy(handle)
	if err != nil {
		return nil, err
	}

	return &protocol.DestroyResponse{}, nil
}

func (s *WardenServer) handleList(list *protocol.ListRequest) (proto.Message, error) {
	containers, err := s.backend.Containers()
	if err != nil {
		return nil, err
	}

	handles := []string{}

	for _, container := range containers {
		handles = append(handles, container.Handle())
	}

	return &protocol.ListResponse{Handles: handles}, nil
}

func (s *WardenServer) handleCopyOut(copyOut *protocol.CopyOutRequest) (proto.Message, error) {
	handle := copyOut.GetHandle()
	srcPath := copyOut.GetSrcPath()
	dstPath := copyOut.GetDstPath()
	owner := copyOut.GetOwner()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	err = container.CopyOut(srcPath, dstPath, owner)
	if err != nil {
		return nil, err
	}

	return &protocol.CopyOutResponse{}, nil
}

func (s *WardenServer) handleStop(request *protocol.StopRequest) (proto.Message, error) {
	handle := request.GetHandle()
	kill := request.GetKill()
	background := request.GetBackground()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	if background {
		go container.Stop(kill)
	} else {
		err = container.Stop(kill)
		if err != nil {
			return nil, err
		}
	}

	return &protocol.StopResponse{}, nil
}

func (s *WardenServer) handleCopyIn(copyIn *protocol.CopyInRequest) (proto.Message, error) {
	handle := copyIn.GetHandle()
	srcPath := copyIn.GetSrcPath()
	dstPath := copyIn.GetDstPath()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	err = container.CopyIn(srcPath, dstPath)
	if err != nil {
		return nil, err
	}

	return &protocol.CopyInResponse{}, nil
}

func (s *WardenServer) handleSpawn(spawn *protocol.SpawnRequest) (proto.Message, error) {
	handle := spawn.GetHandle()
	script := spawn.GetScript()
	privileged := spawn.GetPrivileged()
	discardOutput := spawn.GetDiscardOutput()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	jobSpec := backend.JobSpec{
		Script:        script,
		Privileged:    privileged,
		DiscardOutput: discardOutput,
	}

	jobID, err := container.Spawn(jobSpec)
	if err != nil {
		return nil, err
	}

	return &protocol.SpawnResponse{JobId: proto.Uint32(jobID)}, nil
}

func (s *WardenServer) handleLink(link *protocol.LinkRequest) (proto.Message, error) {
	handle := link.GetHandle()
	jobID := link.GetJobId()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	jobResult, err := container.Link(jobID)
	if err != nil {
		return nil, err
	}

	return &protocol.LinkResponse{
		ExitStatus: proto.Uint32(jobResult.ExitStatus),
		Stdout:     proto.String(string(jobResult.Stdout)),
		Stderr:     proto.String(string(jobResult.Stderr)),
	}, nil
}

func (s *WardenServer) handleRun(request *protocol.RunRequest) (proto.Message, error) {
	handle := request.GetHandle()
	script := request.GetScript()
	privileged := request.GetPrivileged()
	discardOutput := request.GetDiscardOutput()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	jobSpec := backend.JobSpec{
		Script:        script,
		Privileged:    privileged,
		DiscardOutput: discardOutput,
	}

	jobID, err := container.Spawn(jobSpec)
	if err != nil {
		return nil, err
	}

	jobResult, err := container.Link(jobID)
	if err != nil {
		return nil, err
	}

	return &protocol.RunResponse{
		ExitStatus: proto.Uint32(jobResult.ExitStatus),
		Stdout:     proto.String(string(jobResult.Stdout)),
		Stderr:     proto.String(string(jobResult.Stderr)),
	}, nil
}

func (s *WardenServer) handleLimitBandwidth(request *protocol.LimitBandwidthRequest) (proto.Message, error) {
	handle := request.GetHandle()
	rate := request.GetRate()
	burst := request.GetBurst()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	limits, err := container.LimitBandwidth(backend.BandwidthLimits{
		RateInBytesPerSecond:      rate,
		BurstRateInBytesPerSecond: burst,
	})

	if err != nil {
		return nil, err
	}

	return &protocol.LimitBandwidthResponse{
		Rate:  proto.Uint64(limits.RateInBytesPerSecond),
		Burst: proto.Uint64(limits.BurstRateInBytesPerSecond),
	}, nil
}

func (s *WardenServer) handleLimitMemory(request *protocol.LimitMemoryRequest) (proto.Message, error) {
	handle := request.GetHandle()
	limitInBytes := request.GetLimitInBytes()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	limits, err := container.LimitMemory(backend.MemoryLimits{
		LimitInBytes: limitInBytes,
	})

	if err != nil {
		return nil, err
	}

	return &protocol.LimitMemoryResponse{
		LimitInBytes: proto.Uint64(limits.LimitInBytes),
	}, nil
}

func (s *WardenServer) handleLimitDisk(request *protocol.LimitDiskRequest) (proto.Message, error) {
	handle := request.GetHandle()
	blockSoft := request.GetBlockSoft()
	blockHard := request.GetBlockHard()
	inodeSoft := request.GetInodeSoft()
	inodeHard := request.GetInodeHard()
	byteSoft := request.GetByteSoft()
	byteHard := request.GetByteHard()

	if request.Block != nil {
		blockHard = request.GetBlock()
	}

	if request.BlockLimit != nil {
		blockHard = request.GetBlockLimit()
	}

	if request.Inode != nil {
		inodeHard = request.GetInode()
	}

	if request.InodeLimit != nil {
		inodeHard = request.GetInodeLimit()
	}

	if request.Byte != nil {
		byteHard = request.GetByte()
	}

	if request.ByteLimit != nil {
		byteHard = request.GetByteLimit()
	}

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	limits, err := container.LimitDisk(backend.DiskLimits{
		BlockSoft: blockSoft,
		BlockHard: blockHard,
		InodeSoft: inodeSoft,
		InodeHard: inodeHard,
		ByteSoft:  byteSoft,
		ByteHard:  byteHard,
	})

	if err != nil {
		return nil, err
	}

	return &protocol.LimitDiskResponse{
		BlockSoft: proto.Uint64(limits.BlockSoft),
		BlockHard: proto.Uint64(limits.BlockHard),
		InodeSoft: proto.Uint64(limits.InodeSoft),
		InodeHard: proto.Uint64(limits.InodeHard),
		ByteSoft:  proto.Uint64(limits.ByteSoft),
		ByteHard:  proto.Uint64(limits.ByteHard),
	}, nil
}

func (s *WardenServer) handleNetIn(request *protocol.NetInRequest) (proto.Message, error) {
	handle := request.GetHandle()
	hostPort := request.GetHostPort()
	containerPort := request.GetContainerPort()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	hostPort, containerPort, err = container.NetIn(hostPort, containerPort)
	if err != nil {
		return nil, err
	}

	return &protocol.NetInResponse{
		HostPort:      proto.Uint32(hostPort),
		ContainerPort: proto.Uint32(containerPort),
	}, nil
}

func (s *WardenServer) handleNetOut(request *protocol.NetOutRequest) (proto.Message, error) {
	handle := request.GetHandle()
	network := request.GetNetwork()
	port := request.GetPort()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	err = container.NetOut(network, port)
	if err != nil {
		return nil, err
	}

	return &protocol.NetOutResponse{}, nil
}

func (s *WardenServer) handleStream(conn net.Conn, request *protocol.StreamRequest) (proto.Message, error) {
	handle := request.GetHandle()
	jobID := request.GetJobId()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	stream, err := container.Stream(jobID)
	if err != nil {
		return nil, err
	}

	var response proto.Message

	for chunk := range stream {
		if chunk.ExitStatus != nil {
			response = &protocol.StreamResponse{
				ExitStatus: proto.Uint32(*chunk.ExitStatus),
			}

			break
		}

		protocol.Messages(&protocol.StreamResponse{
			Name: proto.String(chunk.Name),
			Data: proto.String(string(chunk.Data)),
		}).WriteTo(conn)
	}

	return response, nil
}

func (s *WardenServer) handleInfo(request *protocol.InfoRequest) (proto.Message, error) {
	handle := request.GetHandle()

	container, err := s.backend.Lookup(handle)
	if err != nil {
		return nil, err
	}

	info, err := container.Info()
	if err != nil {
		return nil, err
	}

	jobIDs := make([]uint64, len(info.JobIDs))
	for i, jobID := range info.JobIDs {
		jobIDs[i] = uint64(jobID)
	}

	return &protocol.InfoResponse{
		State:         proto.String(info.State),
		Events:        info.Events,
		HostIp:        proto.String(info.HostIP),
		ContainerIp:   proto.String(info.ContainerIP),
		ContainerPath: proto.String(info.ContainerPath),
		JobIds:        jobIDs,

		MemoryStat: &protocol.InfoResponse_MemoryStat{
			Cache:                   proto.Uint64(info.MemoryStat.Cache),
			Rss:                     proto.Uint64(info.MemoryStat.Rss),
			MappedFile:              proto.Uint64(info.MemoryStat.MappedFile),
			Pgpgin:                  proto.Uint64(info.MemoryStat.Pgpgin),
			Pgpgout:                 proto.Uint64(info.MemoryStat.Pgpgout),
			Swap:                    proto.Uint64(info.MemoryStat.Swap),
			Pgfault:                 proto.Uint64(info.MemoryStat.Pgfault),
			Pgmajfault:              proto.Uint64(info.MemoryStat.Pgmajfault),
			InactiveAnon:            proto.Uint64(info.MemoryStat.InactiveAnon),
			ActiveAnon:              proto.Uint64(info.MemoryStat.ActiveAnon),
			InactiveFile:            proto.Uint64(info.MemoryStat.InactiveFile),
			ActiveFile:              proto.Uint64(info.MemoryStat.ActiveFile),
			Unevictable:             proto.Uint64(info.MemoryStat.Unevictable),
			HierarchicalMemoryLimit: proto.Uint64(info.MemoryStat.HierarchicalMemoryLimit),
			HierarchicalMemswLimit:  proto.Uint64(info.MemoryStat.HierarchicalMemswLimit),
			TotalCache:              proto.Uint64(info.MemoryStat.TotalCache),
			TotalRss:                proto.Uint64(info.MemoryStat.TotalRss),
			TotalMappedFile:         proto.Uint64(info.MemoryStat.TotalMappedFile),
			TotalPgpgin:             proto.Uint64(info.MemoryStat.TotalPgpgin),
			TotalPgpgout:            proto.Uint64(info.MemoryStat.TotalPgpgout),
			TotalSwap:               proto.Uint64(info.MemoryStat.TotalSwap),
			TotalPgfault:            proto.Uint64(info.MemoryStat.TotalPgfault),
			TotalPgmajfault:         proto.Uint64(info.MemoryStat.TotalPgmajfault),
			TotalInactiveAnon:       proto.Uint64(info.MemoryStat.TotalInactiveAnon),
			TotalActiveAnon:         proto.Uint64(info.MemoryStat.TotalActiveAnon),
			TotalInactiveFile:       proto.Uint64(info.MemoryStat.TotalInactiveFile),
			TotalActiveFile:         proto.Uint64(info.MemoryStat.TotalActiveFile),
			TotalUnevictable:        proto.Uint64(info.MemoryStat.TotalUnevictable),
		},

		CpuStat: &protocol.InfoResponse_CpuStat{
			Usage:  proto.Uint64(info.CPUStat.Usage),
			User:   proto.Uint64(info.CPUStat.User),
			System: proto.Uint64(info.CPUStat.System),
		},

		DiskStat: &protocol.InfoResponse_DiskStat{
			BytesUsed:  proto.Uint64(info.DiskStat.BytesUsed),
			InodesUsed: proto.Uint64(info.DiskStat.InodesUsed),
		},

		BandwidthStat: &protocol.InfoResponse_BandwidthStat{
			InRate:   proto.Uint64(info.BandwidthStat.InRate),
			InBurst:  proto.Uint64(info.BandwidthStat.InBurst),
			OutRate:  proto.Uint64(info.BandwidthStat.OutRate),
			OutBurst: proto.Uint64(info.BandwidthStat.OutBurst),
		},
	}, nil
}
