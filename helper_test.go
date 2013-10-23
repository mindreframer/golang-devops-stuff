package router

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"testing"
	"time"

	steno "github.com/cloudfoundry/gosteno"
	. "launchpad.net/gocheck"

	"github.com/cloudfoundry/gorouter/config"
)

func Test(t *testing.T) {
	config := &steno.Config{
		Sinks: []steno.Sink{},
		Codec: steno.NewJsonCodec(),
		Level: steno.LOG_INFO,
	}

	steno.Init(config)

	// log = steno.NewLogger("test")

	TestingT(t)
}

func SpecConfig(natsPort, statusPort, proxyPort uint16) *config.Config {
	c := config.DefaultConfig()

	c.Port = proxyPort
	c.Index = 2
	c.TraceKey = "my_trace_key"

	// Hardcode the IP to localhost to avoid leaving the machine while running tests
	c.Ip = "127.0.0.1"

	c.StartResponseDelayInterval = 10 * time.Millisecond
	c.PublishStartMessageIntervalInSeconds = 10
	c.PruneStaleDropletsInterval = 0
	c.DropletStaleThreshold = 0
	c.PublishActiveAppsInterval = 0

	c.EndpointTimeout = 500 * time.Millisecond

	c.Status = config.StatusConfig{
		Port: statusPort,
		User: "user",
		Pass: "pass",
	}

	c.Nats = config.NatsConfig{
		Host: "localhost",
		Port: natsPort,
		User: "nats",
		Pass: "nats",
	}

	c.Logging = config.LoggingConfig{
		File:  "/dev/stderr",
		Level: "info",
	}

	return c
}

func StartNats(port int) *exec.Cmd {
	cmd := exec.Command("nats-server", "-p", strconv.Itoa(port), "--user", "nats", "--pass", "nats")
	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("NATS failed to start: %v\n", err))
	}

	return cmd
}

func StopNats(cmd *exec.Cmd) {
	cmd.Process.Kill()
	cmd.Wait()
}

func nextAvailPort() uint16 {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	defer listener.Close()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		panic(err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}

	return uint16(port)
}

func waitUntilNatsUp(port uint16) error {
	maxWait := 10
	for i := 0; i < maxWait; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return nil
		}
	}

	return errors.New("Waited too long for NATS to start")
}

func waitUntilNatsDown(port uint16) error {
	maxWait := 10
	for i := 0; i < maxWait; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return nil
		}
	}

	return errors.New("Waited too long for NATS to stop")
}
