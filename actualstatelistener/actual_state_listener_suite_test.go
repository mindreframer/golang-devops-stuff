package actualstatelistener_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestActualStateListener(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Actual State Listener Suite")
}
