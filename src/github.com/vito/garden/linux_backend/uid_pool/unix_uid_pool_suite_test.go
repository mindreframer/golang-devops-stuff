package uid_pool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUid_pool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unix UID Pool Suite")
}
