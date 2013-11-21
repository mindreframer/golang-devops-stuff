package evacuator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEvacuator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Evacuator Suite")
}
