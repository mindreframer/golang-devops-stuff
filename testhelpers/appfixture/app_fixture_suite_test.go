package appfixture_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAppFixture(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Fixture Suite")
}
