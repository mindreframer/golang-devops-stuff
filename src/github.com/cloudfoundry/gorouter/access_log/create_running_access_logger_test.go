package access_log_test

import (
	. "github.com/cloudfoundry/gorouter/access_log"

	"github.com/cloudfoundry/gorouter/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AccessLog", func() {

	It("creates null access loger if no access log and no loggregregator url", func() {
		config := config.DefaultConfig()
		Ω(CreateRunningAccessLogger(config)).To(BeAssignableToTypeOf(&NullAccessLogger{}))
	})

	It("creates an access log when loggegrator url specified", func() {
		config := config.DefaultConfig()
		config.LoggregatorConfig.Url = "10.10.3.13:4325"
		config.AccessLog = ""

		Ω(CreateRunningAccessLogger(config)).To(BeAssignableToTypeOf(&FileAndLoggregatorAccessLogger{}))
	})

	It("creates an access log if an access log is specified", func() {
		config := config.DefaultConfig()
		config.AccessLog = "/dev/null"

		Ω(CreateRunningAccessLogger(config)).To(BeAssignableToTypeOf(&FileAndLoggregatorAccessLogger{}))
	})

	It("creates an AccessLogger if both access log and loggregator url are specififed", func() {
		config := config.DefaultConfig()
		config.LoggregatorConfig.Url = "10.10.3.13:4325"
		config.AccessLog = "/dev/null"

		Ω(CreateRunningAccessLogger(config)).To(BeAssignableToTypeOf(&FileAndLoggregatorAccessLogger{}))
	})

	It("reports an error if the access log location is invalid", func() {
		config := config.DefaultConfig()
		config.AccessLog = "/this\\is/illegal"

		a, err := CreateRunningAccessLogger(config)
		Ω(err).To(HaveOccurred())
		Ω(a).To(BeNil())
	})
})
