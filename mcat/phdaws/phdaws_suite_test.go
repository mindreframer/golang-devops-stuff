package phd_aws

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPhdAWS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "MCAT AWS PhD Suite", []Reporter{&DataReporter{Title: "Local_ETCD"}})
}
