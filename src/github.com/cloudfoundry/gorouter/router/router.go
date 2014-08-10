package router

import (
	"github.com/cloudfoundry-incubator/dropsonde/autowire"
	vcap "github.com/cloudfoundry/gorouter/common"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/proxy"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/varz"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"

	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var DrainTimeout = errors.New("router: Drain timeout")

type Router struct {
	config     *config.Config
	proxy      proxy.Proxy
	mbusClient *yagnats.Client
	registry   *registry.RouteRegistry
	varz       varz.Varz
	component  *vcap.VcapComponent

	listener net.Listener

	logger *steno.Logger
}

func NewRouter(cfg *config.Config, p proxy.Proxy, mbusClient *yagnats.Client, r *registry.RouteRegistry, v varz.Varz,
	logCounter *vcap.LogCounter) (*Router, error) {

	var host string
	if cfg.Status.Port != 0 {
		host = fmt.Sprintf("%s:%d", cfg.Ip, cfg.Status.Port)
	}

	varz := &vcap.Varz{
		UniqueVarz: v,
		GenericVarz: vcap.GenericVarz{
			LogCounts: logCounter,
		},
	}

	healthz := &vcap.Healthz{}

	component := &vcap.VcapComponent{
		Type:        "Router",
		Index:       cfg.Index,
		Host:        host,
		Credentials: []string{cfg.Status.User, cfg.Status.Pass},
		Config:      cfg,
		Varz:        varz,
		Healthz:     healthz,
		InfoRoutes: map[string]json.Marshaler{
			"/routes": r,
		},
	}

	router := &Router{
		config:     cfg,
		proxy:      p,
		mbusClient: mbusClient,
		registry:   r,
		varz:       v,
		component:  component,
		logger:     steno.NewLogger("router"),
	}

	if err := router.component.Start(); err != nil {
		return nil, err
	}

	return router, nil
}

func (r *Router) Run() <-chan error {
	r.registry.StartPruningCycle()

	r.RegisterComponent()

	// Subscribe register/unregister router
	r.SubscribeRegister()
	r.HandleGreetings()
	r.SubscribeUnregister()

	// Kickstart sending start messages
	r.SendStartMessage()

	// Send start again on reconnect
	r.mbusClient.ConnectedCallback = func() {
		r.SendStartMessage()
	}

	// Schedule flushing active app's app_id
	r.ScheduleFlushApps()

	// Wait for one start message send interval, such that the router's registry
	// can be populated before serving requests.
	if r.config.StartResponseDelayInterval != 0 {
		r.logger.Infof("Waiting %s before listening...", r.config.StartResponseDelayInterval)
		time.Sleep(r.config.StartResponseDelayInterval)
	}

	server := http.Server{
		Handler: autowire.InstrumentedHandler(r.proxy),
	}

	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.config.Port))
	if err != nil {
		r.logger.Fatalf("net.Listen: %s", err)
		errChan <- err
		return errChan
	}

	r.listener = listener
	r.logger.Infof("Listening on %s", listener.Addr())

	go func() {
		err := server.Serve(listener)
		errChan <- err
	}()

	return errChan
}

func (r *Router) Drain(drainTimeout time.Duration) error {
	r.listener.Close()

	drained := make(chan struct{})
	go func() {
		r.proxy.Wait()
		close(drained)
	}()

	select {
	case <-drained:
	case <-time.After(drainTimeout):
		r.logger.Warn("router.drain.timed-out")
		return DrainTimeout
	}
	return nil
}

func (r *Router) Stop() {
	r.listener.Close()
	r.component.Stop()
}

func (r *Router) RegisterComponent() {
	r.component.Register(r.mbusClient)
}

func (r *Router) SubscribeRegister() {
	r.subscribeRegistry("router.register", func(registryMessage *registryMessage) {
		r.logger.Debugf("Got router.register: %v", registryMessage)

		for _, uri := range registryMessage.Uris {
			r.registry.Register(
				uri,
				registryMessage.makeEndpoint(),
			)
		}
	})
}

func (r *Router) SubscribeUnregister() {
	r.subscribeRegistry("router.unregister", func(registryMessage *registryMessage) {
		r.logger.Debugf("Got router.unregister: %v", registryMessage)

		for _, uri := range registryMessage.Uris {
			r.registry.Unregister(
				uri,
				registryMessage.makeEndpoint(),
			)
		}
	})
}

func (r *Router) HandleGreetings() {
	r.mbusClient.Subscribe("router.greet", func(msg *yagnats.Message) {
		response, _ := r.greetMessage()
		r.mbusClient.Publish(msg.ReplyTo, response)
	})
}

func (r *Router) SendStartMessage() {
	b, err := r.greetMessage()
	if err != nil {
		panic(err)
	}

	// Send start message once at start
	r.mbusClient.Publish("router.start", b)
}

func (r *Router) ScheduleFlushApps() {
	if r.config.PublishActiveAppsInterval == 0 {
		return
	}

	go func() {
		t := time.NewTicker(r.config.PublishActiveAppsInterval)
		x := time.Now()

		for {
			select {
			case <-t.C:
				y := time.Now()
				r.flushApps(x)
				x = y
			}
		}
	}()
}

func (r *Router) flushApps(t time.Time) {
	x := r.varz.ActiveApps().ActiveSince(t)

	y, err := json.Marshal(x)
	if err != nil {
		r.logger.Warnf("flushApps: Error marshalling JSON: %s", err)
		return
	}

	b := bytes.Buffer{}
	w := zlib.NewWriter(&b)
	w.Write(y)
	w.Close()

	z := b.Bytes()

	r.logger.Debugf("Active apps: %d, message size: %d", len(x), len(z))

	r.mbusClient.Publish("router.active_apps", z)
}

func (r *Router) greetMessage() ([]byte, error) {
	host, err := vcap.LocalIP()
	if err != nil {
		return nil, err
	}

	d := vcap.RouterStart{
		Id:    r.component.UUID,
		Hosts: []string{host},
		MinimumRegisterIntervalInSeconds: r.config.StartResponseDelayIntervalInSeconds,
	}

	return json.Marshal(d)
}

func (r *Router) subscribeRegistry(subject string, successCallback func(*registryMessage)) {
	callback := func(message *yagnats.Message) {
		payload := message.Payload

		var msg registryMessage

		err := json.Unmarshal(payload, &msg)
		if err != nil {
			logMessage := fmt.Sprintf("%s: Error unmarshalling JSON (%d; %s): %s", subject, len(payload), payload, err)
			r.logger.Warnd(map[string]interface{}{"payload": string(payload)}, logMessage)
		}

		logMessage := fmt.Sprintf("%s: Received message", subject)
		r.logger.Debugd(map[string]interface{}{"message": msg}, logMessage)

		successCallback(&msg)
	}

	_, err := r.mbusClient.Subscribe(subject, callback)
	if err != nil {
		r.logger.Errorf("Error subscribing to %s: %s", subject, err)
	}
}
