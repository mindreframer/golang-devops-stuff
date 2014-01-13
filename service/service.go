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
	"github.com/mailgun/vulcan/control/js"
	"github.com/mailgun/vulcan/discovery"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/loadbalance/roundrobin"
	"github.com/mailgun/vulcan/metrics"
	"github.com/mailgun/vulcan/timeutils"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"time"
)

type Service struct {
	options *serviceOptions
	proxy   *vulcan.ReverseProxy
	metrics metrics.ProxyMetrics
}

// Initializes service from the command line args
func NewService() (*Service, error) {
	options, err := parseOptions()
	if err != nil {
		return nil, err
	}
	return &Service{options: options, metrics: metrics.NewProxyMetrics()}, nil
}

// This is a blocking call, starts reverse proxy, connects to the backends, etc
func (s *Service) Start() error {
	if s.options.cpuProfile != "" {
		f, err := os.Create(s.options.cpuProfile)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				glog.Errorf("captured %v, stopping profiler and exiting..", sig)
				pprof.StopCPUProfile()
				os.Exit(1)
			}
		}()
	}

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

	outputs := strings.Split(s.options.metricsOutput, ",")
	for _, v := range outputs {
		metrics.AddOutput(v)
	}

	if s.options.sslCertFile != "" && s.options.sslKeyFile == "" {
		return nil, fmt.Errorf("invalid configuration: -sslkey unspecified, but -sslcert was specified.")
	} else if s.options.sslCertFile == "" && s.options.sslKeyFile != "" {
		return nil, fmt.Errorf("invalid configuration: -sslcert unspecified, but -sslkey was specified.")
	}

	var discoveryService discovery.Service

	if s.options.discovery != "" {
		discoveryUrl := s.options.discovery
		if s.options.discovery == "etcd" {
			// TODO remove this compat hack?
			discoveryUrl = "etcd://" + strings.Join(s.options.etcdEndpoints, ",")
			discoveryService = discovery.NewEtcd(s.options.etcdEndpoints)
		}

		discoveryService, err = discovery.New(discoveryUrl)
		if err != nil {
			return nil, err
		}
	}

	controller := &js.JsController{
		CodeGetter:       js.NewFileGetter(s.options.codePath),
		DiscoveryService: discoveryService,
	}

	proxySettings := &vulcan.ProxySettings{
		Controller:       controller,
		ThrottlerBackend: b,
		LoadBalancer:     loadBalancer,
	}

	proxy, err := vulcan.NewReverseProxy(&s.metrics, proxySettings)
	if err != nil {
		return nil, err
	}
	controller.Client = proxy
	return proxy, nil
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

	if s.options.sslCertFile != "" && s.options.sslKeyFile != "" {
		return server.ListenAndServeTLS(s.options.sslCertFile, s.options.sslKeyFile)
	}

	return server.ListenAndServe()
}
