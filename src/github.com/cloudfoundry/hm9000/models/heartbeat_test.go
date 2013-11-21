package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeat", func() {
	var heartbeat Heartbeat

	BeforeEach(func() {
		heartbeat = Heartbeat{
			DeaGuid: "dea_abc",
			InstanceHeartbeats: []InstanceHeartbeat{
				{
					AppGuid:        "abc",
					AppVersion:     "xyz-123",
					InstanceGuid:   "def",
					InstanceIndex:  3,
					State:          InstanceStateRunning,
					StateTimestamp: 1123.2,
					DeaGuid:        "dea_abc",
				},
			},
		}
	})

	Describe("Building from JSON", func() {
		Context("When all is well", func() {
			It("should, like, totally build from JSON", func() {
				jsonHeartbeat, err := NewHeartbeatFromJSON([]byte(`{
                    "dea":"dea_abc",
                    "droplets":[
                        {
                            "cc_partition":"default",
                            "droplet":"abc",
                            "version":"xyz-123",
                            "instance":"def",
                            "index":3,
                            "state":"RUNNING",
                            "state_timestamp":1123.2
                        }
                    ]
                }`))

				Ω(err).ShouldNot(HaveOccured())

				Ω(jsonHeartbeat).Should(Equal(heartbeat))
			})
		})

		Context("When the JSON is invalid", func() {
			It("returns a zero heartbeat and an error", func() {
				heartbeat, err := NewHeartbeatFromJSON([]byte(`{`))

				Ω(heartbeat).Should(BeZero())
				Ω(err).Should(HaveOccured())
			})
		})
	})

	Describe("ToJson", func() {
		It("should, like, totally encode JSON", func() {
			jsonHeartbeat, err := NewHeartbeatFromJSON(heartbeat.ToJSON())

			Ω(err).ShouldNot(HaveOccured())
			Ω(jsonHeartbeat).Should(Equal(heartbeat))
		})
	})

	Context("With a complex heartbeat", func() {
		var heartbeat Heartbeat
		var app appfixture.AppFixture
		BeforeEach(func() {
			app = appfixture.NewAppFixture()

			crashedHeartbeat := app.InstanceAtIndex(2).Heartbeat()
			crashedHeartbeat.State = InstanceStateCrashed

			startingHeartbeat := app.InstanceAtIndex(3).Heartbeat()
			startingHeartbeat.State = InstanceStateStarting

			evacuatingHeartbeat := app.InstanceAtIndex(4).Heartbeat()
			evacuatingHeartbeat.State = InstanceStateEvacuating

			heartbeat = Heartbeat{
				DeaGuid: "abc",
				InstanceHeartbeats: []InstanceHeartbeat{
					crashedHeartbeat,
					startingHeartbeat,
					evacuatingHeartbeat,
					app.InstanceAtIndex(0).Heartbeat(),
				},
			}
		})

		Describe("LogDescription", func() {
			It("should return a nice rollup", func() {
				desc := heartbeat.LogDescription()
				Ω(desc["DEA"]).Should(Equal("abc"))
				Ω(desc["Evacuating"]).Should(Equal("1"))
				Ω(desc["Crashed"]).Should(Equal("1"))
				Ω(desc["Starting"]).Should(Equal("1"))
				Ω(desc["Running"]).Should(Equal("1"))
			})
		})
	})
})
