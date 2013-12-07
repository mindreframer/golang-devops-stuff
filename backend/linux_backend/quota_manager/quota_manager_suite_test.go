package quota_manager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuota_manager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Quota Manager Suite")
}
