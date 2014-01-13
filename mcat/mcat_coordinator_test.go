package mcat_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/desiredstateserver"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/natsrunner"
	"github.com/cloudfoundry/hm9000/testhelpers/startstoplistener"
	"github.com/cloudfoundry/hm9000/testhelpers/storerunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/gomega"
	"strconv"
	"time"
)

type MCATCoordinator struct {
	MessageBus   yagnats.NATSClient
	StateServer  *desiredstateserver.DesiredStateServer
	StoreRunner  storerunner.StoreRunner
	StoreAdapter storeadapter.StoreAdapter

	natsRunner        *natsrunner.NATSRunner
	startStopListener *startstoplistener.StartStopListener

	Conf *config.Config

	CurrentStoreType          string
	DesiredStateServerBaseUrl string
	DesiredStateServerPort    int
	NatsPort                  int
	MetricsServerPort         int

	ParallelNode int
	Verbose      bool

	currentCLIRunner *CLIRunner
}

func NewMCATCoordinator(parallelNode int, verbose bool) *MCATCoordinator {
	coordinator := &MCATCoordinator{
		ParallelNode: parallelNode,
		Verbose:      verbose,
	}
	coordinator.loadConfig()
	coordinator.computePorts()

	return coordinator
}

func (coordinator *MCATCoordinator) loadConfig() {
	conf, err := config.DefaultConfig()
	Ω(err).ShouldNot(HaveOccurred())
	coordinator.Conf = conf
}

func (coordinator *MCATCoordinator) computePorts() {
	coordinator.DesiredStateServerPort = 6001 + coordinator.ParallelNode
	coordinator.DesiredStateServerBaseUrl = "http://127.0.0.1:" + strconv.Itoa(coordinator.DesiredStateServerPort)
	coordinator.NatsPort = 4223 + coordinator.ParallelNode
	coordinator.MetricsServerPort = 7879 + coordinator.ParallelNode
}

func (coordinator *MCATCoordinator) PrepForNextTest() (*CLIRunner, *Simulator, *startstoplistener.StartStopListener) {
	coordinator.StoreRunner.Reset()
	coordinator.startStopListener.Reset()
	coordinator.StateServer.Reset()

	if coordinator.currentCLIRunner != nil {
		coordinator.currentCLIRunner.Cleanup()
	}
	coordinator.currentCLIRunner = NewCLIRunner(coordinator.CurrentStoreType, coordinator.StoreRunner.NodeURLS(), coordinator.DesiredStateServerBaseUrl, coordinator.NatsPort, coordinator.MetricsServerPort, coordinator.Verbose)
	store := storepackage.NewStore(coordinator.Conf, coordinator.StoreAdapter, fakelogger.NewFakeLogger())
	simulator := NewSimulator(coordinator.Conf, coordinator.StoreRunner, store, coordinator.StateServer, coordinator.currentCLIRunner, coordinator.MessageBus)

	return coordinator.currentCLIRunner, simulator, coordinator.startStopListener
}

func (coordinator *MCATCoordinator) StartNats() {
	coordinator.natsRunner = natsrunner.NewNATSRunner(coordinator.NatsPort)
	coordinator.natsRunner.Start()
	coordinator.MessageBus = coordinator.natsRunner.MessageBus
}

func (coordinator *MCATCoordinator) StartDesiredStateServer() {
	coordinator.StateServer = desiredstateserver.NewDesiredStateServer()
	go coordinator.StateServer.SpinUp(coordinator.DesiredStateServerPort)
}

func (coordinator *MCATCoordinator) StartStartStopListener() {
	coordinator.startStopListener = startstoplistener.NewStartStopListener(coordinator.MessageBus, coordinator.Conf)
}

func (coordinator *MCATCoordinator) StartETCD() {
	coordinator.CurrentStoreType = "etcd"
	etcdPort := 5000 + (coordinator.ParallelNode-1)*10
	coordinator.StoreRunner = storerunner.NewETCDClusterRunner(etcdPort, 1)
	coordinator.StoreRunner.Start()

	coordinator.StoreAdapter = storeadapter.NewETCDStoreAdapter(coordinator.StoreRunner.NodeURLS(), workerpool.NewWorkerPool(coordinator.Conf.StoreMaxConcurrentRequests))
	err := coordinator.StoreAdapter.Connect()
	Ω(err).ShouldNot(HaveOccurred())
}

func (coordinator *MCATCoordinator) StartZooKeeper() {
	coordinator.CurrentStoreType = "ZooKeeper"
	zookeeperPort := 2181 + (coordinator.ParallelNode-1)*10
	coordinator.StoreRunner = storerunner.NewZookeeperClusterRunner(zookeeperPort, 1)
	coordinator.StoreRunner.Start()

	coordinator.StoreAdapter = storeadapter.NewZookeeperStoreAdapter(coordinator.StoreRunner.NodeURLS(), workerpool.NewWorkerPool(coordinator.Conf.StoreMaxConcurrentRequests), &timeprovider.RealTimeProvider{}, time.Second)
	err := coordinator.StoreAdapter.Connect()
	Ω(err).ShouldNot(HaveOccurred())
}

func (coordinator *MCATCoordinator) StopStore() {
	coordinator.StoreRunner.Stop()
	if coordinator.StoreAdapter != nil {
		coordinator.StoreAdapter.Disconnect()
	}
}

func (coordinator *MCATCoordinator) StopAllExternalProcesses() {
	coordinator.StoreRunner.Stop()
	coordinator.natsRunner.Stop()

	if coordinator.currentCLIRunner != nil {
		coordinator.currentCLIRunner.Cleanup()
	}
}
