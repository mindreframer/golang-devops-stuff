package lifecycle_test

import (
	"log"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/integration/garden_runner"
	"github.com/vito/gordon"
)

var runner *garden_runner.GardenRunner
var client gordon.Client

func TestLifecycle(t *testing.T) {
	remote := os.Getenv("GARDEN_REMOTE_HOST")

	var rootPath, rootFSPath string

	if remote != "" {
		rootPath = "/vagrant/root"
		rootFSPath = "/opt/warden/rootfs"
	} else {
		rootPath = "../../root"
		rootFSPath = os.Getenv("GARDEN_TEST_ROOTFS")
	}

	if rootFSPath == "" {
		log.Println("GARDEN_TEST_ROOTFS undefined; skipping")
		return
	}

	var err error

	runner, err = garden_runner.New(rootPath, rootFSPath, remote)
	if err != nil {
		log.Fatalln("failed to create runner:", err)
	}

	err = runner.Start()
	if err != nil {
		log.Fatalln("garden failed to start:", err)
	}

	client = runner.NewClient()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle Suite")

	err = runner.Stop()
	if err != nil {
		log.Fatalln("garden failed to stop:", err)
	}

	err = runner.TearDown()
	if err != nil {
		log.Fatalln("failed to tear down server:", err)
	}
}
