package desiredstatefetcher_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDesiredStateFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Desired State Fetcher Suite")
}
