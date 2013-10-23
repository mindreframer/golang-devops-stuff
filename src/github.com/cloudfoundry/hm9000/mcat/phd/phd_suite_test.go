package phd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/testhelpers/storerunner"

	"os"
	"os/signal"
	"testing"
)

var storeRunner storerunner.StoreRunner

func TestPhd(t *testing.T) {
	registerSignalHandler()
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t, "MCAT PhD Performance Suite", []Reporter{&DataReporter{}})

	if storeRunner != nil {
		storeRunner.Stop()
	}
}

func registerSignalHandler() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		select {
		case <-c:
			if storeRunner != nil {
				storeRunner.Stop()
			}
			os.Exit(0)
		}
	}()
}
