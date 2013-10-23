package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"time"
)

var _ = Describe("DesiredAppState", func() {

	Describe("JSON", func() {
		var desiredAppState DesiredAppState

		BeforeEach(func() {
			desiredAppState = DesiredAppState{
				AppGuid:           "app_guid_abc",
				AppVersion:        "app_version_123",
				NumberOfInstances: 3,
				Memory:            1024,
				State:             AppStateStopped,
				PackageState:      AppPackageStateStaged,
				UpdatedAt:         time.Unix(0, 0),
			}
		})

		Describe("loading JSON", func() {
			Context("When all is well", func() {
				It("should, like, totally build from JSON", func() {
					timeAsJson, _ := desiredAppState.UpdatedAt.MarshalJSON()
					json := fmt.Sprintf(`{
	                    "id":"app_guid_abc",
	                    "version":"app_version_123",
	                    "instances":3,
	                    "memory":1024,
	                    "state":"STOPPED",
	                    "package_state":"STAGED",
	                    "updated_at":%s
	                }`, timeAsJson)
					jsonDesired, err := NewDesiredAppStateFromJSON([]byte(json))

					Ω(err).ShouldNot(HaveOccured())

					Ω(jsonDesired).Should(EqualDesiredState(desiredAppState))
				})
			})

			Context("When the JSON is invalid", func() {
				It("returns a zero desired state and an error", func() {
					desired, err := NewDesiredAppStateFromJSON([]byte(`{`))

					Ω(desired).Should(BeZero())
					Ω(err).Should(HaveOccured())
				})
			})
		})

		Describe("writing JSON", func() {
			It("outputs to JSON", func() {
				var decoded DesiredAppState
				err := json.Unmarshal(desiredAppState.ToJSON(), &decoded)
				Ω(err).ShouldNot(HaveOccured())
				Ω(decoded).Should(EqualDesiredState(desiredAppState))
			})
		})
	})

	Describe("StoreKey", func() {
		var appstate DesiredAppState

		BeforeEach(func() {
			appstate = DesiredAppState{
				AppGuid:    "XYZ-ABC",
				AppVersion: "DEF-123",
			}
		})

		It("returns the key for the store", func() {
			Ω(appstate.StoreKey()).Should(Equal("XYZ-ABC-DEF-123"))
		})
	})

	Describe("Equality", func() {
		var (
			actual DesiredAppState
			other  DesiredAppState
		)
		BeforeEach(func() {
			actual = DesiredAppState{
				AppGuid:           "a guid",
				AppVersion:        "a version",
				NumberOfInstances: 1,
				Memory:            256,
				State:             AppStateStarted,
				PackageState:      AppPackageStateStaged,
				UpdatedAt:         time.Unix(0, 0),
			}

			other = actual
		})

		It("is equal when all fields are equal", func() {
			Ω(actual.Equal(other)).Should(BeTrue())
		})

		It("is inequal when the app guid is different", func() {
			other.AppGuid = "not an app guid"
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the app version is different", func() {
			other.AppVersion = "not an app version"
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the number of instances is different", func() {
			other.NumberOfInstances = 9000
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the memory is different", func() {
			other.Memory = 4096
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the state is different", func() {
			other.State = AppStateStopped
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the package state is different", func() {
			other.PackageState = AppPackageStateFailed
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the updated at is different", func() {
			other.UpdatedAt = time.Unix(9000, 9000)
			Ω(actual.Equal(other)).Should(BeFalse())
		})
	})

	Describe("LogDescription", func() {
		var desiredAppState DesiredAppState

		BeforeEach(func() {
			desiredAppState = DesiredAppState{
				AppGuid:           "app_guid_abc",
				AppVersion:        "app_version_123",
				NumberOfInstances: 3,
				Memory:            1024,
				State:             AppStateStopped,
				PackageState:      AppPackageStateStaged,
				UpdatedAt:         time.Unix(10, 0),
			}
		})

		It("should return correct message", func() {
			Ω(desiredAppState.LogDescription()).Should(Equal(map[string]string{
				"AppGuid":           "app_guid_abc",
				"AppVersion":        "app_version_123",
				"NumberOfInstances": "3",
				"Memory":            "1024",
				"State":             "STOPPED",
				"PackageState":      "STAGED",
				"UpdatedAt":         "10",
			}))
		})
	})
})
