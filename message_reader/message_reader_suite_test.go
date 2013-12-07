package message_reader_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMessagereader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Messagereader Suite")
}
