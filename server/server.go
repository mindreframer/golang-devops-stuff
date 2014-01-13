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
	"github.com/vito/garden/drain"
	"github.com/vito/garden/message_reader"
	protocol "github.com/vito/garden/protocol"
	"github.com/vito/garden/server/bomberman"
)

type WardenServer struct {
	socketPath         string
	containerGraceTime time.Duration
	backend            backend.Backend

	listener     net.Listener
	openRequests *drain.Drain

	setStopping chan bool
	stopping    chan bool

	bomberman *bomberman.Bomberman
}

type UnhandledRequestError struct {
	Request proto.Message
}

func (e UnhandledRequestError) Error() string {
	return fmt.Sprintf("unhandled request type: %T", e.Request)
}

func New(
	socketPath string,
	containerGraceTime time.Duration,
	backend backend.Backend,
) *WardenServer {
	return &WardenServer{
		socketPath:         socketPath,
		containerGraceTime: containerGraceTime,
		backend:            backend,

		setStopping: make(chan bool),
		stopping:    make(chan bool),

		openRequests: drain.New(),
	}
}

func (s *WardenServer) Start() error {
	err := s.removeExistingSocket()
	if err != nil {
		return err
	}

	err = s.backend.Start()
	if err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}

	s.listener = listener

	os.Chmod(s.socketPath, 0777)

	containers, err := s.backend.Containers()
	if err != nil {
		return err
	}

	s.bomberman = bomberman.New(s.reapContainer)

	for _, container := range containers {
		s.bomberman.Strap(container)
	}

	go s.trackStopping()
	go s.handleConnections(listener)

	return nil
}

func (s *WardenServer) Stop() {
	s.setStopping <- true
	s.listener.Close()
	s.openRequests.Wait()
	s.backend.Stop()
}

func (s *WardenServer) trackStopping() {
	stopping := false

	for {
		select {
		case stopping = <-s.setStopping:
		case s.stopping <- stopping:
		}
	}
}

func (s *WardenServer) handleConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// listener closed
			break
		}

		go s.serveConnection(conn)
	}
}

func (s *WardenServer) serveConnection(conn net.Conn) {
	read := bufio.NewReader(conn)

	for {
		var response proto.Message
		var err error

		if <-s.stopping {
			conn.Close()
			break
		}

		request, err := message_reader.ReadRequest(read)
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("error reading request:", err)
			continue
		}

		if <-s.stopping {
			conn.Close()
			break
		}

		s.openRequests.Incr()

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
			s.openRequests.Decr()
			response, err = s.handleLink(request.(*protocol.LinkRequest))
			s.openRequests.Incr()
		case *protocol.StreamRequest:
			s.openRequests.Decr()
			response, err = s.handleStream(conn, request.(*protocol.StreamRequest))
			s.openRequests.Incr()
		case *protocol.RunRequest:
			s.openRequests.Decr()
			response, err = s.handleRun(request.(*protocol.RunRequest))
			s.openRequests.Incr()
		case *protocol.LimitBandwidthRequest:
			response, err = s.handleLimitBandwidth(request.(*protocol.LimitBandwidthRequest))
		case *protocol.LimitMemoryRequest:
			response, err = s.handleLimitMemory(request.(*protocol.LimitMemoryRequest))
		case *protocol.LimitDiskRequest:
			response, err = s.handleLimitDisk(request.(*protocol.LimitDiskRequest))
		case *protocol.LimitCpuRequest:
			response, err = s.handleLimitCpu(request.(*protocol.LimitCpuRequest))
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

		s.openRequests.Decr()
	}
}

func (s *WardenServer) removeExistingSocket() error {
	if _, err := os.Stat(s.socketPath); os.IsNotExist(err) {
		return nil
	}

	err := os.Remove(s.socketPath)

	if err != nil {
		return fmt.Errorf("error deleting existing socket: %s", err)
	}

	return nil
}

func (s *WardenServer) reapContainer(container backend.Container) {
	log.Printf("reaping %s (idle for %s)\n", container.Handle(), container.GraceTime())
	s.backend.Destroy(container.Handle())
}
