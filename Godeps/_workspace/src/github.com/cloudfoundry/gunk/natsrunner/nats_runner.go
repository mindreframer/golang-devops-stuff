package natsrunner

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var natsCommand *exec.Cmd

type NATSRunner struct {
	port        int
	natsSession *gexec.Session
	MessageBus  yagnats.NATSClient
}

func NewNATSRunner(port int) *NATSRunner {
	return &NATSRunner{
		port: port,
	}
}

func (runner *NATSRunner) Start() {
	if runner.natsSession != nil {
		panic("starting an already started NATS runner!!!")
	}

	_, err := exec.LookPath("gnatsd")
	if err != nil {
		fmt.Println("You need gnatsd installed!")
		os.Exit(1)
	}

	cmd := exec.Command("gnatsd", "-p", strconv.Itoa(runner.port))
	sess, err := gexec.Start(
		cmd,
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[34m[gnatsd]\x1b[0m ", ginkgo.GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[34m[gnatsd]\x1b[0m ", ginkgo.GinkgoWriter),
	)
	Ω(err).ShouldNot(HaveOccurred(), "Make sure to have gnatsd on your path")

	runner.natsSession = sess

	connectionInfo := &yagnats.ConnectionInfo{
		Addr: fmt.Sprintf("127.0.0.1:%d", runner.port),
	}

	messageBus := yagnats.NewClient()

	Eventually(func() error {
		return messageBus.Connect(connectionInfo)
	}, 5, 0.1).ShouldNot(HaveOccurred())

	runner.MessageBus = messageBus
}

func (runner *NATSRunner) Stop() {
	runner.KillWithFire()
}

func (runner *NATSRunner) KillWithFire() {
	if runner.natsSession != nil {
		runner.natsSession.Kill().Wait(5 * time.Second)
		runner.MessageBus = nil
		runner.natsSession = nil
	}
}
