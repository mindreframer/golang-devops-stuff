package lifecycle_test

import (
	"log"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/gordon"

	"github.com/vito/garden/integration/garden_runner"
)

var runner *garden_runner.GardenRunner
var client *gordon.Client

func TestLifecycle(t *testing.T) {
	rootPath := "../../root"
	rootFSPath := os.Getenv("GARDEN_TEST_ROOTFS")

	if rootFSPath == "" {
		log.Println("GARDEN_TEST_ROOTFS undefined; skipping")
		return
	}

	var err error

	runner, err = garden_runner.New(rootPath, rootFSPath)
	if err != nil {
		log.Fatalln("failed to create runner:", err)
	}

	err = runner.Start()
	if err != nil {
		log.Fatalln("garden failed to start:", err)
	}

	client = gordon.NewClient(&gordon.ConnectionInfo{
		SocketPath: runner.SocketPath,
	})

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
