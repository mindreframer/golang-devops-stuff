package access_log_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAccessLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AccessLog Suite")
}
