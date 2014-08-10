package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"
	. "github.com/cloudfoundry/gorouter/common/http"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var procStat *ProcessStatus

type VcapComponent struct {
	// These fields are from individual components
	Type        string                    `json:"type"`
	Index       uint                      `json:"index"`
	Host        string                    `json:"host"`
	Credentials []string                  `json:"credentials"`
	Config      interface{}               `json:"-"`
	Varz        *Varz                     `json:"-"`
	Healthz     *Healthz                  `json:"-"`
	InfoRoutes  map[string]json.Marshaler `json:"-"`
	Logger      *steno.Logger             `json:"-"`

	// These fields are automatically generated
	UUID      string   `json:"uuid"`
	StartTime Time     `json:"start"`
	Uptime    Duration `json:"uptime"`

	listener net.Listener
	statusCh chan error
	quitCh   chan struct{}
}

type RouterStart struct {
	Id                               string   `json:"id"`
	Hosts                            []string `json:"hosts"`
	MinimumRegisterIntervalInSeconds int      `json:"minimumRegisterIntervalInSeconds"`
}

func (c *VcapComponent) UpdateVarz() {
	c.Varz.Lock()
	defer c.Varz.Unlock()

	procStat.RLock()
	c.Varz.MemStat = procStat.MemRss
	c.Varz.Cpu = procStat.CpuUsage
	procStat.RUnlock()
	c.Varz.Uptime = c.StartTime.Elapsed()
}

func (c *VcapComponent) Start() error {
	if c.Type == "" {
		log.Error("Component type is required")
		return errors.New("type is required")
	}

	c.quitCh = make(chan struct{}, 1)
	c.StartTime = Time(time.Now())
	uuid, err := GenerateUUID()
	if err != nil {
		return err
	}
	c.UUID = fmt.Sprintf("%d-%s", c.Index, uuid)

	if c.Host == "" {
		host, err := LocalIP()
		if err != nil {
			log.Error(err.Error())
			return err
		}

		port, err := GrabEphemeralPort()
		if err != nil {
			log.Error(err.Error())
			return err
		}

		c.Host = fmt.Sprintf("%s:%d", host, port)
	}

	if c.Credentials == nil || len(c.Credentials) != 2 {
		user, err := GenerateUUID()
		if err != nil {
			return err
		}
		password, err := GenerateUUID()
		if err != nil {
			return err
		}

		c.Credentials = []string{user, password}
	}

	if c.Logger != nil {
		log = c.Logger
	}

	c.Varz.NumCores = runtime.NumCPU()
	c.Varz.component = *c

	procStat = NewProcessStatus()

	c.ListenAndServe()
	return nil
}

func (c *VcapComponent) Register(mbusClient yagnats.NATSClient) error {
	mbusClient.Subscribe("vcap.component.discover", func(msg *yagnats.Message) {
		c.Uptime = c.StartTime.Elapsed()
		b, e := json.Marshal(c)
		if e != nil {
			log.Warnf(e.Error())
			return
		}

		mbusClient.Publish(msg.ReplyTo, b)
	})

	b, e := json.Marshal(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}

	mbusClient.Publish("vcap.component.announce", b)

	log.Infof("Component %s registered successfully", c.Type)
	return nil
}

func (c *VcapComponent) Stop() {
	close(c.quitCh)
	if c.listener != nil {
		c.listener.Close()
		<-c.statusCh
	}
}

func (c *VcapComponent) ListenAndServe() {
	hs := http.NewServeMux()

	hs.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, c.Healthz.Value())
	})

	hs.HandleFunc("/varz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		c.UpdateVarz()
		enc.Encode(c.Varz)
	})

	for path, marshaler := range c.InfoRoutes {
		hs.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			enc := json.NewEncoder(w)
			enc.Encode(marshaler)
		})
	}

	f := func(user, password string) bool {
		return user == c.Credentials[0] && password == c.Credentials[1]
	}

	s := &http.Server{
		Addr:         c.Host,
		Handler:      &BasicAuth{hs, f},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	c.statusCh = make(chan error, 1)
	l, err := net.Listen("tcp", c.Host)
	if err != nil {
		c.statusCh <- err
		return
	}
	c.listener = l

	go func() {
		err = s.Serve(l)
		select {
		case <-c.quitCh:
			c.statusCh <- nil

		default:
			c.statusCh <- err
		}
	}()
}
