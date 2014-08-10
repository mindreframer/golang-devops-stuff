package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var gorouterPath string

func TestGorouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gorouter Suite")
}

var _ = BeforeSuite(func() {
	path, err := gexec.Build("github.com/cloudfoundry/gorouter", "-race")
	Ω(err).ShouldNot(HaveOccurred())
	gorouterPath = path
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
