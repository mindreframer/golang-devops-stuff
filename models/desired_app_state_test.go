package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
)

var _ = Describe("DesiredAppState", func() {

	Describe("JSON", func() {
		var desiredAppState DesiredAppState

		BeforeEach(func() {
			desiredAppState = DesiredAppState{
				AppGuid:           "app_guid_abc",
				AppVersion:        "app_version_123",
				NumberOfInstances: 3,
				State:             AppStateStopped,
				PackageState:      AppPackageStateStaged,
			}
		})

		Describe("loading JSON", func() {
			Context("When all is well", func() {
				It("should, like, totally build from JSON", func() {
					jsonDesired, err := NewDesiredAppStateFromJSON([]byte(`{
	                    "id":"app_guid_abc",
	                    "version":"app_version_123",
	                    "instances":3,
	                    "state":"STOPPED",
	                    "package_state":"STAGED"
	                }`))

					Ω(err).ShouldNot(HaveOccurred())

					Ω(jsonDesired).Should(EqualDesiredState(desiredAppState))
				})
			})

			Context("When the JSON is invalid", func() {
				It("returns a zero desired state and an error", func() {
					desired, err := NewDesiredAppStateFromJSON([]byte(`{`))

					Ω(desired).Should(BeZero())
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Describe("writing JSON", func() {
			It("outputs to JSON", func() {
				var decoded DesiredAppState
				err := json.Unmarshal(desiredAppState.ToJSON(), &decoded)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(decoded).Should(EqualDesiredState(desiredAppState))
			})
		})

		Describe("loading CSV", func() {
			Context("When all is well", func() {
				It("should, like, totally build from CSV", func() {
					csvDesired, err := NewDesiredAppStateFromCSV("app_guid_abc", "app_version_123", []byte("3,STOPPED,STAGED"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(csvDesired).Should(EqualDesiredState(desiredAppState))
				})
			})

			Context("When the CSV is invalid", func() {
				It("returns a zero desired state and an error", func() {
					desired, err := NewDesiredAppStateFromCSV("app_guid_abc", "app_version_123", []byte(`1,STOPPED`))
					Ω(desired).Should(BeZero())
					Ω(err).Should(HaveOccurred())

					desired, err = NewDesiredAppStateFromCSV("app_guid_abc", "app_version_123", []byte(`LOL,STOPPED`))
					Ω(desired).Should(BeZero())
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Describe("writing CSV", func() {
			It("outputs to CSV", func() {
				Ω(string(desiredAppState.ToCSV())).Should(Equal("3,STOPPED,STAGED"))
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
			Ω(appstate.StoreKey()).Should(Equal("XYZ-ABC,DEF-123"))
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
				State:             AppStateStarted,
				PackageState:      AppPackageStateStaged,
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

		It("is inequal when the state is different", func() {
			other.State = AppStateStopped
			Ω(actual.Equal(other)).Should(BeFalse())
		})

		It("is inequal when the package state is different", func() {
			other.PackageState = AppPackageStateFailed
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
				State:             AppStateStopped,
				PackageState:      AppPackageStateStaged,
			}
		})

		It("should return correct message", func() {
			Ω(desiredAppState.LogDescription()).Should(Equal(map[string]string{
				"AppGuid":           "app_guid_abc",
				"AppVersion":        "app_version_123",
				"NumberOfInstances": "3",
				"State":             "STOPPED",
				"PackageState":      "STAGED",
			}))
		})
	})
})
