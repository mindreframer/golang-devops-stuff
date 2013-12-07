package remote_command_runner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRemote_command_runner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Remote_command_runner Suite")
}
