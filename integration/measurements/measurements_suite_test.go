package measurements_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMeasurements(t *testing.T) {
	RegisterFailHandler(Fail)

	if os.Getenv("WARDEN_TEST_SOCKET") == "" {
		fmt.Println("WARDEN_TEST_SOCKET not set; skipping.")
		return
	}

	RunSpecs(t, "Measurements Suite")
}
