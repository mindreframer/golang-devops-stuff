package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	vhost "github.com/inconshreveable/go-vhost"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	muxTimeout            = 10 * time.Second
	defaultConnectTimeout = 10000 // milliseconds
)

type loadTLSConfigFn func(crtPath, keyPath string) (*tls.Config, error)

type Options struct {
	configPath string
}

type Backend struct {
	Addr           string `"yaml:addr"`
	ConnectTimeout int    `yaml:connect_timeout"`
}

type Frontend struct {
	Backends []Backend `yaml:"backends"`
	Strategy string    `yaml:"strategy"`
	TLSCrt   string    `yaml:"tls_crt"`
	mux      *vhost.TLSMuxer
	TLSKey   string `yaml:"tls_key"`
	Default  bool   `yaml:"default"`

	strategy  BackendStrategy `yaml:"-"`
	tlsConfig *tls.Config     `yaml:"-"`
}

type Configuration struct {
	BindAddr        string               `yaml:"bind_addr"`
	Frontends       map[string]*Frontend `yaml:"frontends"`
	defaultFrontend *Frontend
}

type Server struct {
	*log.Logger
	*Configuration
	wait sync.WaitGroup

	// these are for easier testing
	mux   *vhost.TLSMuxer
	ready chan int
}

func (s *Server) Run() error {
	// bind a port to handle TLS connections
	l, err := net.Listen("tcp", s.Configuration.BindAddr)
	if err != nil {
		return err
	}
	s.Printf("Serving connections on %v", l.Addr())

	// start muxing on it
	s.mux, err = vhost.NewTLSMuxer(l, muxTimeout)
	if err != nil {
		return err
	}

	// wait for all frontends to finish
	s.wait.Add(len(s.Frontends))

	// setup muxing for each frontend
	for name, front := range s.Frontends {
		fl, err := s.mux.Listen(name)
		if err != nil {
			return err
		}
		go s.runFrontend(name, front, fl)
	}

	// custom error handler so we can log errors
	go func() {
		for {
			conn, err := s.mux.NextError()

			if conn == nil {
				s.Printf("Failed to mux next connection, error: %v", err)
				if _, ok := err.(vhost.Closed); ok {
					return
				} else {
					continue
				}
			} else {
				if _, ok := err.(vhost.NotFound); ok && s.defaultFrontend != nil {
					go s.proxyConnection(conn, s.defaultFrontend)
				} else {
					s.Printf("Failed to mux connection from %v, error: %v", conn.RemoteAddr(), err)
					// XXX: respond with valid TLS close messages
					conn.Close()
				}
			}
		}
	}()

	// we're ready, signal it for testing
	if s.ready != nil {
		close(s.ready)
	}

	s.wait.Wait()

	return nil
}

func (s *Server) runFrontend(name string, front *Frontend, l net.Listener) {
	// mark finished when done so Run() can return
	defer s.wait.Done()

	// always round-robin strategy for now
	front.strategy = &RoundRobinStrategy{backends: front.Backends}

	s.Printf("Handling connections to %v", name)
	for {
		// accept next connection to this frontend
		conn, err := l.Accept()
		if err != nil {
			s.Printf("Failed to accept new connection for '%v': %v", conn.RemoteAddr())
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					continue
				}
			}
			return
		}
		s.Printf("Accepted new connection for %v from %v", name, conn.RemoteAddr())

		// proxy the connection to an backend
		go s.proxyConnection(conn, front)
	}
}

func (s *Server) proxyConnection(c net.Conn, front *Frontend) (err error) {
	// unwrap if tls cert/key was specified
	if front.tlsConfig != nil {
		c = tls.Server(c, front.tlsConfig)
	}

	// pick the backend
	backend := front.strategy.NextBackend()

	// dial the backend
	upConn, err := net.DialTimeout("tcp", backend.Addr, time.Duration(backend.ConnectTimeout)*time.Millisecond)
	if err != nil {
		s.Printf("Failed to dial backend connection %v: %v", backend.Addr, err)
		c.Close()
		return
	}
	s.Printf("Initiated new connection to backend: %v %v", upConn.LocalAddr(), upConn.RemoteAddr())

	// join the connections
	s.joinConnections(c, upConn)
	return
}

func (s *Server) joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	halfJoin := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		n, err := io.Copy(dst, src)
		s.Printf("Copy from %v to %v failed after %d bytes with error %v", src.RemoteAddr(), dst.RemoteAddr(), n, err)
	}

	s.Printf("Joining connections: %v %v", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(c1, c2)
	go halfJoin(c2, c1)
	wg.Wait()
}

type BackendStrategy interface {
	NextBackend() Backend
}

type RoundRobinStrategy struct {
	backends []Backend
	idx      int
}

func (s *RoundRobinStrategy) NextBackend() Backend {
	n := len(s.backends)

	if n == 1 {
		return s.backends[0]
	} else {
		s.idx = (s.idx + 1) % n
		return s.backends[s.idx]
	}
}

func parseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <config file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s is a simple TLS reverse proxy that can multiplex TLS connections\n"+
			"by inspecting the SNI extension on each incoming connection. This\n"+
			"allows you to accept connections to many different backend TLS\n"+
			"applications on a single port.\n\n"+
			"%s takes a single argument: the path to a YAML configuration file.\n\n", os.Args[0], os.Args[0])
	}
	flag.Parse()

	if len(flag.Args()) != 1 {
		return nil, fmt.Errorf("You must specify a single argument, the path to the configuration file.")
	}

	return &Options{
		configPath: flag.Arg(0),
	}, nil

}

func parseConfig(configBuf []byte, loadTLS loadTLSConfigFn) (config *Configuration, err error) {
	// deserialize/parse the config
	config = new(Configuration)
	if err = goyaml.Unmarshal(configBuf, &config); err != nil {
		err = fmt.Errorf("Error parsing configuration file: %v", err)
		return
	}

	// configuration validation / normalization
	if config.BindAddr == "" {
		err = fmt.Errorf("You must specify a bind_addr")
		return
	}

	if len(config.Frontends) == 0 {
		err = fmt.Errorf("You must specify at least one frontend")
		return
	}

	for name, front := range config.Frontends {
		if len(front.Backends) == 0 {
			err = fmt.Errorf("You must specify at least one backend for frontend '%v'", name)
			return
		}

		if front.Default {
			if config.defaultFrontend != nil {
				err = fmt.Errorf("Only one frontend may be the default")
				return
			}
			config.defaultFrontend = front
		}

		for _, back := range front.Backends {
			if back.ConnectTimeout == 0 {
				back.ConnectTimeout = defaultConnectTimeout
			}

			if back.Addr == "" {
				err = fmt.Errorf("You must specify an addr for each backend on frontend '%v'", name)
				return
			}
		}

		if front.TLSCrt != "" || front.TLSKey != "" {
			if front.tlsConfig, err = loadTLS(front.TLSCrt, front.TLSKey); err != nil {
				err = fmt.Errorf("Failed to load TLS configuration for frontend '%v': %v", name, err)
				return
			}
		}
	}

	return
}

func loadTLSConfig(crtPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

func main() {
	// parse command line options
	opts, err := parseArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// read configuration file
	configBuf, err := ioutil.ReadFile(opts.configPath)
	if err != nil {
		fmt.Printf("Failed to read configuration file %s: %v\n", opts.configPath, err)
		os.Exit(1)
	}

	// parse configuration file
	config, err := parseConfig(configBuf, loadTLSConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// run server
	s := &Server{
		Configuration: config,
		Logger:        log.New(os.Stdout, "slt ", log.LstdFlags|log.Lshortfile),
	}

	// this blocks unless there's a startup error
	err = s.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
		os.Exit(1)
	}
}
