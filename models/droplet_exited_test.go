package models_test

import (
	"encoding/json"
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DropletExited", func() {
	var dropletExited DropletExited

	BeforeEach(func() {
		dropletExited = DropletExited{
			CCPartition:     "default",
			AppGuid:         "app_guid_abc",
			AppVersion:      "app_version_123",
			InstanceGuid:    "instance_guid_xyz",
			InstanceIndex:   1,
			Reason:          DropletExitedReasonStopped,
			ExitStatusCode:  2,
			ExitDescription: "tried to make two parallel lines intersect",
			CrashTimestamp:  3,
		}
	})

	Describe("JSON", func() {
		Describe("loading JSON", func() {
			Context("When all is well", func() {
				It("should, like, totally build from JSON", func() {
					json := `{
                        "cc_partition":"default",
                        "droplet":"app_guid_abc",
                        "version":"app_version_123",
                        "instance":"instance_guid_xyz",
                        "index":1,
                        "reason":"STOPPED",
                        "exit_status":2,
                        "exit_description":"tried to make two parallel lines intersect",
                        "crash_timestamp":3
                    }`
					decoded, err := NewDropletExitedFromJSON([]byte(json))

					Ω(err).ShouldNot(HaveOccured())

					Ω(decoded).Should(Equal(dropletExited))
				})
			})

			Context("When the JSON is invalid", func() {
				It("returns a zero desired state and an error", func() {
					desired, err := NewDropletExitedFromJSON([]byte(`{`))

					Ω(desired).Should(BeZero())
					Ω(err).Should(HaveOccured())
				})
			})
		})

		Describe("writing JSON", func() {
			It("outputs to JSON", func() {
				var decoded DropletExited
				err := json.Unmarshal(dropletExited.ToJSON(), &decoded)
				Ω(err).ShouldNot(HaveOccured())
				Ω(decoded).Should(Equal(dropletExited))
			})
		})
	})

	Describe("LogDescription", func() {
		It("should return the correct message", func() {
			Ω(dropletExited.LogDescription()).Should(Equal(map[string]string{
				"AppGuid":         "app_guid_abc",
				"AppVersion":      "app_version_123",
				"InstanceGuid":    "instance_guid_xyz",
				"InstanceIndex":   "1",
				"Reason":          "STOPPED",
				"ExitStatusCode":  "2",
				"ExitDescription": "tried to make two parallel lines intersect",
			}))
		})
	})
})
