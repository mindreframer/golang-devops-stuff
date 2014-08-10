package autowire_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"log"
	"testing"
)

func TestAutowire(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Autowire Suite")
}
