/*
Contains everything that is needed to run the program:
* Creating PID
* Setting up logging
* Configuring proxy from command line args
*/

package service

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/glogutils"
	"github.com/mailgun/gocql"
	"github.com/mailgun/vulcan"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/control/servicecontrol"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/loadbalance/roundrobin"
	"github.com/mailgun/vulcan/timeutils"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Service struct {
	options *serviceOptions
	proxy   *vulcan.ReverseProxy
}

// Initializes service from the command line args
func NewService() (*Service, error) {
	options, err := parseOptions()
	if err != nil {
		return nil, err
	}
	return &Service{options: options}, nil
}

// This is a blocking call, starts reverse proxy, connects to the backends, etc
func (s *Service) Start() error {
	if err := s.writePid(); err != nil {
		return err
	}
	if err := s.startLogsCleanup(); err != nil {
		return err
	}

	proxy, err := s.initProxy()
	if err != nil {
		return err
	}
	s.proxy = proxy
	return s.startProxy()
}

// Write process id to a file, if asked. This is extremely useful for various monitoring tools
func (s *Service) writePid() error {
	if s.options.pidPath != "" {
		pidBytes := []byte(fmt.Sprintf("%d", os.Getpid()))
		return ioutil.WriteFile(s.options.pidPath, pidBytes, 0644)
	}
	return nil
}

// This function starts cleaning up after glog library, periodically removing logs
// that are no longer used
func (s *Service) startLogsCleanup() error {
	if glogutils.LogDir() != "" {
		glog.Infof("Starting log cleanup go routine with period: %s", s.options.cleanupPeriod)
		go func() {
			t := time.Tick(s.options.cleanupPeriod)
			for {
				select {
				case <-t:
					glog.Infof("Start cleaning up the logs")
					err := glogutils.CleanupLogs()
					if err != nil {
						glog.Errorf("Failed to clean up the logs: %s, shutting down goroutine", err)
						return
					}
				}
			}
		}()
	}
	return nil
}

func (s *Service) initProxy() (*vulcan.ReverseProxy, error) {
	var b backend.Backend
	var err error

	if s.options.backend == "cassandra" {
		cassandraConfig := &backend.CassandraConfig{
			Servers:       s.options.cassandraServers,
			Keyspace:      s.options.cassandraKeyspace,
			Consistency:   gocql.One,
			LaunchCleanup: s.options.cassandraCleanup,
			CleanupTime:   s.options.cassandraCleanupOptions.T,
		}
		b, err = backend.NewCassandraBackend(
			cassandraConfig, &timeutils.RealTime{})
		if err != nil {
			return nil, err
		}
	} else if s.options.backend == "memory" {
		b, err = backend.NewMemoryBackend(&timeutils.RealTime{})
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Unsupported backend")
	}

	var loadBalancer loadbalance.Balancer
	if s.options.loadBalancer == "roundrobin" || s.options.loadBalancer == "random" {
		loadBalancer = roundrobin.NewRoundRobin(&timeutils.RealTime{})
	} else {
		return nil, fmt.Errorf("Unsupported loadbalancing algo")
	}

	controllerSettings := &servicecontrol.Settings{
		Servers:      s.options.controlServers,
		LoadBalancer: loadBalancer,
	}
	controller, err := servicecontrol.NewClient(controllerSettings)
	if err != nil {
		return nil, err
	}

	proxySettings := &vulcan.ProxySettings{
		Controller:       controller,
		ThrottlerBackend: b,
		LoadBalancer:     loadBalancer,
	}

	return vulcan.NewReverseProxy(proxySettings)
}

func (s *Service) startProxy() error {
	addr := fmt.Sprintf("%s:%d", s.options.host, s.options.httpPort)
	server := &http.Server{
		Addr:           addr,
		Handler:        s.proxy,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return server.ListenAndServe()
}
