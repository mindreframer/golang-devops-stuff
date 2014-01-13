package locker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/testhelpers/storerunner"
	"os"
	"os/signal"
	"testing"
)

var etcdRunner storerunner.StoreRunner

func TestLocker(t *testing.T) {
	RegisterFailHandler(Fail)

	etcdRunner = storerunner.NewETCDClusterRunner(5001, 1)

	etcdRunner.Start()

	RunSpecs(t, "Locker Suite")

	etcdRunner.Stop()
}

var _ = BeforeEach(func() {
	etcdRunner.Reset()
})

func registerSignalHandler() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		select {
		case <-c:
			etcdRunner.Stop()
			os.Exit(0)
		}
	}()
}
