package desiredstatefetcher_test

import (
	"github.com/cloudfoundry/hm9000/testhelpers/desiredstateserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

var stateServer *desiredstateserver.DesiredStateServer

func TestDesiredStateFetcher(t *testing.T) {
	stateServer = desiredstateserver.NewDesiredStateServer()
	go stateServer.SpinUp(6001)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Desired State Fetcher Suite")
}
