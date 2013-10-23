package apiserver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApiserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apiserver Suite")
}
