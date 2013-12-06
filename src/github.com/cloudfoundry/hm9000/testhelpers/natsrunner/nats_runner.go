package natsrunner

import (
	"fmt"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/yagnats"

	"os/exec"
	"strconv"
)

var natsCommand *exec.Cmd

type NATSRunner struct {
	port        int
	natsCommand *exec.Cmd
	MessageBus  yagnats.NATSClient
}

func NewNATSRunner(port int) *NATSRunner {
	return &NATSRunner{
		port: port,
	}
}

func (runner *NATSRunner) Start() {
	runner.natsCommand = exec.Command("gnatsd", "-p", strconv.Itoa(runner.port))
	err := runner.natsCommand.Start()
	Ω(err).ShouldNot(HaveOccured(), "Make sure to have gnatsd on your path")

	connectionInfo := &yagnats.ConnectionInfo{
		Addr: fmt.Sprintf("127.0.0.1:%d", runner.port),
	}

	messageBus := yagnats.NewClient()

	Eventually(func() error {
		return messageBus.Connect(connectionInfo)
	}, 5, 0.1).ShouldNot(HaveOccured())

	runner.MessageBus = messageBus
}

func (runner *NATSRunner) Stop() {
	if runner.natsCommand != nil {
		runner.natsCommand.Process.Kill()
		runner.MessageBus = nil
		runner.natsCommand = nil
	}
}
