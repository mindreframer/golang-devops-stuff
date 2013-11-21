package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InstanceHeartbeat", func() {
	var instance InstanceHeartbeat

	BeforeEach(func() {
		instance = InstanceHeartbeat{
			AppGuid:        "abc",
			AppVersion:     "xyz-123",
			InstanceGuid:   "def",
			InstanceIndex:  3,
			State:          InstanceStateRunning,
			StateTimestamp: 1123.2,
			DeaGuid:        "dea_abc",
		}
	})

	Describe("Building from JSON", func() {
		Context("When all is well", func() {
			It("should, like, totally build from JSON", func() {
				jsonInstance, err := NewInstanceHeartbeatFromJSON([]byte(`{
                    "droplet":"abc",
                    "version":"xyz-123",
                    "instance":"def",
                    "index":3,
                    "state":"RUNNING",
                    "state_timestamp":1123.2,
                    "dea_guid":"dea_abc"
                }`))

				Ω(err).ShouldNot(HaveOccured())

				Ω(jsonInstance).Should(Equal(instance))
			})
		})

		Context("When the JSON is invalid", func() {
			It("returns a zero heartbeat and an error", func() {
				instance, err := NewInstanceHeartbeatFromJSON([]byte(`{`))

				Ω(instance).Should(BeZero())
				Ω(err).Should(HaveOccured())
			})
		})
	})

	Describe("ToJSON", func() {
		It("should, like, totally encode JSON", func() {
			jsonInstance, err := NewInstanceHeartbeatFromJSON(instance.ToJSON())

			Ω(err).ShouldNot(HaveOccured())
			Ω(jsonInstance).Should(Equal(instance))
		})
	})

	Describe("Building from CSV", func() {
		Context("When all is well", func() {
			It("should, like, totally build from CSV", func() {
				jsonInstance, err := NewInstanceHeartbeatFromCSV("abc", "xyz-123", "def", []byte(`3,RUNNING,1123.2,dea_abc`))

				Ω(err).ShouldNot(HaveOccured())

				Ω(jsonInstance).Should(Equal(instance))
			})
		})

		Context("When the CSV is invalid", func() {
			It("returns a zero heartbeat and an error", func() {
				instance, err := NewInstanceHeartbeatFromCSV("abc", "xyz-123", "def", []byte(`3,RUNNING,1123.2`))

				Ω(instance).Should(BeZero())
				Ω(err).Should(HaveOccured())

				instance, err = NewInstanceHeartbeatFromCSV("abc", "xyz-123", "def", []byte(`oops,RUNNING,1123.2,dea_abc`))
				Ω(instance).Should(BeZero())
				Ω(err).Should(HaveOccured())

				instance, err = NewInstanceHeartbeatFromCSV("abc", "xyz-123", "def", []byte(`3,RUNNING,oops,dea_abc`))
				Ω(instance).Should(BeZero())
				Ω(err).Should(HaveOccured())
			})
		})
	})

	Describe("ToCSV", func() {
		It("should, like, totally encode to CSV", func() {
			jsonInstance, err := NewInstanceHeartbeatFromCSV("abc", "xyz-123", "def", instance.ToCSV())

			Ω(err).ShouldNot(HaveOccured())
			Ω(jsonInstance).Should(Equal(instance))
		})
	})

	Describe("StoreKey", func() {
		It("returns the key for the store", func() {
			Ω(instance.StoreKey()).Should(Equal("def"))
		})
	})

	Describe("LogDescription", func() {
		It("should return correct message", func() {
			logDescription := instance.LogDescription()

			Ω(logDescription).Should(Equal(map[string]string{
				"AppGuid":        "abc",
				"AppVersion":     "xyz-123",
				"InstanceGuid":   "def",
				"InstanceIndex":  "3",
				"State":          "RUNNING",
				"StateTimestamp": "1123",
				"DeaGuid":        "dea_abc",
			}))
		})
	})

	Describe("Checking Heartbeat State", func() {
		It("should return the correct answer to IsStarting", func() {
			instance.State = InstanceStateStarting
			Ω(instance.IsStarting()).Should(BeTrue())
			instance.State = InstanceStateRunning
			Ω(instance.IsStarting()).Should(BeFalse())
		})

		It("should return the correct answer to IsRunning", func() {
			instance.State = InstanceStateRunning
			Ω(instance.IsRunning()).Should(BeTrue())
			instance.State = InstanceStateStarting
			Ω(instance.IsRunning()).Should(BeFalse())
		})

		It("should return the correct answer to IsCrashed", func() {
			instance.State = InstanceStateCrashed
			Ω(instance.IsCrashed()).Should(BeTrue())
			instance.State = InstanceStateRunning
			Ω(instance.IsCrashed()).Should(BeFalse())
		})

		It("should return the correct answer to IsEvacuating", func() {
			instance.State = InstanceStateEvacuating
			Ω(instance.IsEvacuating()).Should(BeTrue())
			instance.State = InstanceStateRunning
			Ω(instance.IsEvacuating()).Should(BeFalse())
		})

		It("should return the correct answer to IsStartingOrRunning", func() {
			instance.State = InstanceStateCrashed
			Ω(instance.IsStartingOrRunning()).Should(BeFalse())
			instance.State = InstanceStateRunning
			Ω(instance.IsStartingOrRunning()).Should(BeTrue())
			instance.State = InstanceStateStarting
			Ω(instance.IsStartingOrRunning()).Should(BeTrue())
		})
	})
})
