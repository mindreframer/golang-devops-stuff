package md_test

import (
	"github.com/cloudfoundry/hm9000/testhelpers/startstoplistener"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"os"
	"os/exec"
	"os/signal"
	"testing"
)

var (
	coordinator       *MDCoordinator
	simulator         *Simulator
	cliRunner         *CLIRunner
	startStopListener *startstoplistener.StartStopListener
)

func TestMd(t *testing.T) {
	registerSignalHandler()
	RegisterFailHandler(Fail)

	cmd := exec.Command("go", "install", "github.com/cloudfoundry/hm9000")
	output, err := cmd.CombinedOutput()
	if err != nil {
		println("FAILED TO COMPILE HM9000")
		println(string(output))
		os.Exit(1)
	}

	coordinator = NewMDCoordinator(ginkgoConfig.GinkgoConfig.ParallelNode, ginkgoConfig.DefaultReporterConfig.Verbose)
	coordinator.StartNats()
	coordinator.StartDesiredStateServer()
	coordinator.StartStartStopListener()

	//run the suite for Cassandra...
	coordinator.StartCassandra()
	RunSpecs(t, "MCAT Cassandra MD Suite")
	coordinator.StopStore()

	//run the suite for ETCD...
	coordinator.StartETCD()
	RunSpecs(t, "MCAT ETCD MD Suite")
	coordinator.StopStore()

	//...and then for zookeeper
	coordinator.StartZooKeeper()
	RunSpecs(t, "MCAT ZooKeeper MD Suite")
	coordinator.StopStore()

	coordinator.StopAllExternalProcesses()
}

var _ = BeforeEach(func() {
	cliRunner, simulator, startStopListener = coordinator.PrepForNextTest()
})

func registerSignalHandler() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		select {
		case <-c:
			coordinator.StopAllExternalProcesses()
			os.Exit(0)
		}
	}()
}
