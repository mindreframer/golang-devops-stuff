package cgroups_manager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCgroups_manager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Cgroups Manager Suite")
}
