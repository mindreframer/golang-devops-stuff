package command_runner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCommand_runner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command_runner Suite")
}
