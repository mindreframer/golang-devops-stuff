package job_tracker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJob_tracker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Job Tracker Suite")
}
