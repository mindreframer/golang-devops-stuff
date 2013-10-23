package analyzer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAnalyzer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Analyzer Suite")
}
