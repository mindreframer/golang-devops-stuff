package storecassandra_test

import (
	"github.com/cloudfoundry/hm9000/testhelpers/storerunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"os/signal"
	"testing"
)

var cassandraRunner *storerunner.CassandraClusterRunner

func TestStorecassandra(t *testing.T) {
	registerSignalHandler()
	RegisterFailHandler(Fail)

	cassandraRunner = storerunner.NewCassandraClusterRunner(9042)
	cassandraRunner.Start()
	RunSpecs(t, "Cassandra Store Suite")
	cassandraRunner.Stop()
}

var _ = BeforeEach(func() {
	cassandraRunner.Reset()
})

func registerSignalHandler() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		select {
		case <-c:
			cassandraRunner.Stop()
			os.Exit(0)
		}
	}()
}
