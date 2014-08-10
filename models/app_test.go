package models_test

import (
	"time"

	. "github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {
	var (
		fixture            appfixture.AppFixture
		appGuid            string
		appVersion         string
		desired            DesiredAppState
		instanceHeartbeats []InstanceHeartbeat
		crashCounts        map[int]CrashCount
	)

	instance := func(instanceIndex int) appfixture.Instance {
		return fixture.InstanceAtIndex(instanceIndex)
	}

	heartbeat := func(instanceIndex int, state InstanceState) InstanceHeartbeat {
		hb := instance(instanceIndex).Heartbeat()
		hb.State = state
		return hb
	}

	app := func() *App {
		return NewApp(appGuid, appVersion, desired, instanceHeartbeats, crashCounts)
	}

	BeforeEach(func() {
		fixture = appfixture.NewAppFixture()

		appGuid = fixture.AppGuid
		appVersion = fixture.AppVersion

		desired = DesiredAppState{}
		instanceHeartbeats = []InstanceHeartbeat{}
		crashCounts = make(map[int]CrashCount)
	})

	Describe("LogDescription", func() {
		It("should report the app guid and version", func() {
			Ω(app().LogDescription()["AppGuid"]).Should(Equal(appGuid))
			Ω(app().LogDescription()["AppVersion"]).Should(Equal(appVersion))
		})

		Context("when there is no desired state", func() {
			It("should report that", func() {
				Ω(app().LogDescription()["Desired"]).Should(Equal("None"))
			})
		})

		Context("when there is a desired state", func() {
			It("should report on the desired state", func() {
				desired = fixture.DesiredState(2)
				Ω(app().LogDescription()["Desired"]).Should(ContainSubstring(`"NumberOfInstances":2`))
				Ω(app().LogDescription()["Desired"]).Should(ContainSubstring(`"State":"STARTED"`))
				Ω(app().LogDescription()["Desired"]).Should(ContainSubstring(`"PackageState":"STAGED"`))
			})
		})

		Context("When there are no heartbeats", func() {
			It("should report that", func() {
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(Equal("[]"))
			})
		})

		Context("When there are heartbeats", func() {
			It("should report on them", func() {
				instanceHeartbeats = []InstanceHeartbeat{
					heartbeat(0, InstanceStateStarting),
					heartbeat(1, InstanceStateRunning),
				}
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"InstanceGuid":"%s"`, instance(0).InstanceGuid))
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"State":"STARTING"`))
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"InstanceIndex":0`))
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"InstanceGuid":"%s"`, instance(1).InstanceGuid))
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"State":"RUNNING"`))
				Ω(app().LogDescription()["InstanceHeartbeats"]).Should(ContainSubstring(`"InstanceIndex":1`))
			})
		})

		Context("When there are no crash counts", func() {
			It("should report that", func() {
				Ω(app().LogDescription()["CrashCounts"]).Should(Equal("[]"))
			})
		})

		Context("When there are crash counts", func() {
			It("should report on them", func() {
				crashCounts[1] = CrashCount{
					AppGuid:       appGuid,
					AppVersion:    appVersion,
					InstanceIndex: 2,
					CrashCount:    3,
				}

				Ω(app().LogDescription()["CrashCounts"]).Should(ContainSubstring(`"InstanceIndex":2`))
				Ω(app().LogDescription()["CrashCounts"]).Should(ContainSubstring(`"CrashCount":3`))
			})
		})
	})

	Describe("MarshalJSON", func() {
		It("implements MarshalJSON", func() {
			jsonRepresentation := string(app().ToJSON())
			Ω(jsonRepresentation).Should(ContainSubstring(`"droplet":"%s"`, appGuid))
			Ω(jsonRepresentation).Should(ContainSubstring(`"version":"%s"`, appVersion))
			Ω(jsonRepresentation).Should(ContainSubstring(`"desired":{`))
			Ω(jsonRepresentation).Should(ContainSubstring(`"instance_heartbeats":[`))
			Ω(jsonRepresentation).Should(ContainSubstring(`"crash_counts":[`))
		})
	})

	Describe("ToJSON", func() {
		It("should generate a JSON representation correctly", func() {
			jsonRepresentation := string(app().ToJSON())
			Ω(jsonRepresentation).Should(ContainSubstring(`"droplet":"%s"`, appGuid))
			Ω(jsonRepresentation).Should(ContainSubstring(`"version":"%s"`, appVersion))
			Ω(jsonRepresentation).Should(ContainSubstring(`"desired":{`))
			Ω(jsonRepresentation).Should(ContainSubstring(`"instance_heartbeats":[`))
			Ω(jsonRepresentation).Should(ContainSubstring(`"crash_counts":[`))
		})
	})

	Describe("IsDesired", func() {
		It("should be desired only if the desired state is non-zero", func() {
			Ω(app().IsDesired()).Should(BeFalse())
			desired = fixture.DesiredState(1)
			Ω(app().IsDesired()).Should(BeTrue())
		})
	})

	Describe("NumberOfDesiredInstances", func() {
		It("should return the number in the desired state", func() {
			Ω(app().NumberOfDesiredInstances()).Should(Equal(0))
			desired = fixture.DesiredState(2)
			Ω(app().NumberOfDesiredInstances()).Should(Equal(2))
		})
	})

	Describe("IsStaged", func() {
		Context("when the app is desired", func() {
			BeforeEach(func() {
				desired = fixture.DesiredState(1)
			})
			It("should be true only when the package state is staged", func() {
				desired.PackageState = AppPackageStateStaged
				Ω(app().IsStaged()).Should(BeTrue())

				desired.PackageState = AppPackageStatePending
				Ω(app().IsStaged()).Should(BeFalse())

				desired.PackageState = AppPackageStateFailed
				Ω(app().IsStaged()).Should(BeFalse())
			})
		})

		Context("when the app is not desired", func() {
			It("should be false", func() {
				Ω(app().IsStaged()).Should(BeFalse())
			})
		})
	})

	Describe("IsIndexDesired", func() {
		It("should be true if the index is less than the number of desired instances", func() {
			Ω(app().IsIndexDesired(0)).Should(BeFalse())
			desired = fixture.DesiredState(3)
			Ω(app().IsIndexDesired(0)).Should(BeTrue())
			Ω(app().IsIndexDesired(1)).Should(BeTrue())
			Ω(app().IsIndexDesired(2)).Should(BeTrue())
			Ω(app().IsIndexDesired(3)).Should(BeFalse())
		})
	})

	Describe("InstanceWithGuid", func() {
		It("should return the instance matching the passed in guid", func() {
			Ω(app().InstanceWithGuid("abc")).Should(BeZero())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateRunning),
				heartbeat(1, InstanceStateRunning),
			}
			Ω(app().InstanceWithGuid("abc")).Should(BeZero())
			Ω(app().InstanceWithGuid(instance(0).InstanceGuid)).Should(Equal(heartbeat(0, InstanceStateRunning)))
			Ω(app().InstanceWithGuid(instance(1).InstanceGuid)).Should(Equal(heartbeat(1, InstanceStateRunning)))
		})
	})

	Describe("ExtraStartingOrRunningInstances", func() {
		It("should return any instances outside of the desired range that are STARTING or RUNNING", func() {
			Ω(app().ExtraStartingOrRunningInstances()).Should(BeEmpty())
			desired = fixture.DesiredState(2)
			Ω(app().ExtraStartingOrRunningInstances()).Should(BeEmpty())

			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateRunning),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateRunning),
				heartbeat(3, InstanceStateCrashed),
				heartbeat(4, InstanceStateStarting),
			}

			Ω(app().ExtraStartingOrRunningInstances()).Should(HaveLen(2))
			Ω(app().ExtraStartingOrRunningInstances()).Should(ContainElement(heartbeat(2, InstanceStateRunning)))
			Ω(app().ExtraStartingOrRunningInstances()).Should(ContainElement(heartbeat(4, InstanceStateStarting)))
		})
	})

	Describe("HasStartingOrRunningInstances", func() {
		It("should be true if there are *any* starting/running instances", func() {
			Ω(app().HasStartingOrRunningInstances()).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
			}
			Ω(app().HasStartingOrRunningInstances()).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
			}
			Ω(app().HasStartingOrRunningInstances()).Should(BeTrue())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateStarting),
			}
			Ω(app().HasStartingOrRunningInstances()).Should(BeTrue())
		})
	})

	Describe("NumberOfDesiredIndicesWithAStartingOrRunningInstance", func() {
		It("should return the number of *desired* indices with a starting or running instance", func() {
			Ω(app().NumberOfDesiredIndicesWithAStartingOrRunningInstance()).Should(Equal(0))
			desired = fixture.DesiredState(3)
			Ω(app().NumberOfDesiredIndicesWithAStartingOrRunningInstance()).Should(Equal(0))
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
				heartbeat(3, InstanceStateStarting),
				heartbeat(4, InstanceStateRunning),
			}
			Ω(app().NumberOfDesiredIndicesWithAStartingOrRunningInstance()).Should(Equal(2))
		})
	})

	Describe("StartingOrRunningInstancesAtIndex", func() {
		It("should return the starting/running instances at the passed in index", func() {
			Ω(app().StartingOrRunningInstancesAtIndex(0)).Should(BeEmpty())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
				heartbeat(2, InstanceStateRunning),
			}
			Ω(app().StartingOrRunningInstancesAtIndex(0)).Should(BeEmpty())
			Ω(app().StartingOrRunningInstancesAtIndex(1)).Should(HaveLen(1))
			Ω(app().StartingOrRunningInstancesAtIndex(1)).Should(ContainElement(heartbeat(1, InstanceStateRunning)))
			Ω(app().StartingOrRunningInstancesAtIndex(2)).Should(HaveLen(2))
			Ω(app().StartingOrRunningInstancesAtIndex(2)).Should(ContainElement(heartbeat(2, InstanceStateStarting)))
			Ω(app().StartingOrRunningInstancesAtIndex(2)).Should(ContainElement(heartbeat(2, InstanceStateRunning)))
		})
	})

	Describe("HeartbeatsByIndex", func() {
		It("returns all of the heartbeats grouped by index", func() {
			Ω(app().HeartbeatsByIndex()).Should(BeEmpty())

			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
			}

			Ω(app().HeartbeatsByIndex()).Should(Equal(map[int][]InstanceHeartbeat{
				0: {
					heartbeat(0, InstanceStateCrashed),
				},
				1: {
					heartbeat(1, InstanceStateRunning),
					heartbeat(1, InstanceStateCrashed),
				},
				2: {
					heartbeat(2, InstanceStateStarting),
					heartbeat(2, InstanceStateCrashed),
				},
			}))
		})
	})

	Describe("HasStartingOrRunningInstanceAtIndex", func() {
		It("should return true if there are starting or running instances at the passed in index", func() {
			Ω(app().HasStartingOrRunningInstanceAtIndex(1)).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
				heartbeat(2, InstanceStateRunning),
			}
			Ω(app().HasStartingOrRunningInstanceAtIndex(0)).Should(BeFalse())
			Ω(app().HasStartingOrRunningInstanceAtIndex(1)).Should(BeTrue())
			Ω(app().HasStartingOrRunningInstanceAtIndex(2)).Should(BeTrue())
		})
	})

	Describe("HasStartingInstanceAtIndex", func() {
		It("should return true if there are starting instances at the passed in index", func() {
			Ω(app().HasStartingInstanceAtIndex(1)).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
			}
			Ω(app().HasStartingInstanceAtIndex(0)).Should(BeFalse())
			Ω(app().HasStartingInstanceAtIndex(1)).Should(BeFalse())
			Ω(app().HasStartingInstanceAtIndex(2)).Should(BeTrue())
		})
	})

	Describe("HasRunningInstanceAtIndex", func() {
		It("should return true if there are running instances at the passed in index", func() {
			Ω(app().HasRunningInstanceAtIndex(1)).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
			}
			Ω(app().HasRunningInstanceAtIndex(0)).Should(BeFalse())
			Ω(app().HasRunningInstanceAtIndex(1)).Should(BeTrue())
			Ω(app().HasRunningInstanceAtIndex(2)).Should(BeFalse())
		})
	})

	Describe("EvacuatingInstancesAtIndex", func() {
		It("should return true if there are evacuating instances at the passed in index", func() {
			instances := app().EvacuatingInstancesAtIndex(1)
			Ω(instances).Should(BeEmpty())

			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateEvacuating),
				heartbeat(1, InstanceStateEvacuating),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateCrashed),
			}

			instances = app().EvacuatingInstancesAtIndex(0)
			Ω(instances).Should(BeEmpty())

			instances = app().EvacuatingInstancesAtIndex(1)
			Ω(instances).Should(HaveLen(2))
			Ω(instances[0]).Should(Equal(heartbeat(1, InstanceStateEvacuating)))
			Ω(instances[1]).Should(Equal(heartbeat(1, InstanceStateEvacuating)))
		})
	})

	Describe("HasCrashedInstanceAtIndex", func() {
		It("should return true if there is a crashed instance at the passed in index", func() {
			Ω(app().HasCrashedInstanceAtIndex(1)).Should(BeFalse())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateRunning),
			}
			Ω(app().HasCrashedInstanceAtIndex(0)).Should(BeTrue())
			Ω(app().HasCrashedInstanceAtIndex(1)).Should(BeTrue())
			Ω(app().HasCrashedInstanceAtIndex(2)).Should(BeFalse())
		})
	})

	Describe("CrashCountAtIndex", func() {
		BeforeEach(func() {
			crashCounts[1] = CrashCount{
				AppGuid:       fixture.AppGuid,
				AppVersion:    fixture.AppVersion,
				InstanceIndex: 1,
				CrashCount:    2,
				CreatedAt:     17,
			}
		})
		Context("when there is a crash count for the passed in index", func() {
			It("should return that crash count", func() {
				Ω(app().CrashCountAtIndex(1, time.Unix(120, 0))).Should(Equal(crashCounts[1]))
			})
		})

		Context("when there is no crash count", func() {
			It("should return a correctly configured crash count", func() {
				Ω(app().CrashCountAtIndex(2, time.Unix(120, 0))).Should(Equal(CrashCount{
					AppGuid:       fixture.AppGuid,
					AppVersion:    fixture.AppVersion,
					InstanceIndex: 2,
					CrashCount:    0,
					CreatedAt:     120,
				}))
			})
		})
	})

	Describe("NumberOfDesiredIndicesReporting", func() {
		It("should return the number of desired indices that have at least one heartbeat reporting (regardless of state)", func() {
			Ω(app().NumberOfDesiredIndicesReporting()).Should(Equal(0))
			desired = fixture.DesiredState(4)
			Ω(app().NumberOfDesiredIndicesReporting()).Should(Equal(0))
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(2, InstanceStateStarting),
				heartbeat(4, InstanceStateRunning),
				heartbeat(5, InstanceStateRunning),
				heartbeat(6, InstanceStateRunning),
			}
			Ω(app().NumberOfDesiredIndicesReporting()).Should(Equal(3))
		})
	})

	Describe("NumberOfStartingOrRunningInstances", func() {
		It("should return the number instances in the starting or running state", func() {
			Ω(app().NumberOfStartingOrRunningInstances()).Should(Equal(0))
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(4, InstanceStateRunning),
				heartbeat(5, InstanceStateStarting),
				heartbeat(6, InstanceStateRunning),
			}
			Ω(app().NumberOfStartingOrRunningInstances()).Should(Equal(5))
		})
	})

	Describe("NumberOfCrashedInstances", func() {
		It("should return the number of instances that are in the crashed state", func() {
			Ω(app().NumberOfCrashedInstances()).Should(Equal(0))
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
			}
			Ω(app().NumberOfCrashedInstances()).Should(Equal(2))
		})
	})

	Describe("NumberOfCrashedIndices", func() {
		It("should return the number of indices that have a crashed instance reporting and no starting/running instance", func() {
			Ω(app().NumberOfCrashedIndices()).Should(Equal(0))
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(3, InstanceStateCrashed),
				heartbeat(3, InstanceStateCrashed),
			}
			Ω(app().NumberOfCrashedIndices()).Should(Equal(2))
		})
	})

	Describe("InstanceHeartbeatsAtIndex", func() {
		It("should return the instance heartbeats at the passed in index", func() {
			Ω(app().InstanceHeartbeatsAtIndex(1)).Should(BeEmpty())
			instanceHeartbeats = []InstanceHeartbeat{
				heartbeat(0, InstanceStateCrashed),
				heartbeat(1, InstanceStateRunning),
				heartbeat(1, InstanceStateCrashed),
				heartbeat(2, InstanceStateStarting),
				heartbeat(2, InstanceStateRunning),
			}
			Ω(app().InstanceHeartbeatsAtIndex(0)).Should(HaveLen(1))
			Ω(app().InstanceHeartbeatsAtIndex(0)).Should(ContainElement(heartbeat(0, InstanceStateCrashed)))
			Ω(app().InstanceHeartbeatsAtIndex(1)).Should(HaveLen(2))
			Ω(app().InstanceHeartbeatsAtIndex(1)).Should(ContainElement(heartbeat(1, InstanceStateRunning)))
			Ω(app().InstanceHeartbeatsAtIndex(1)).Should(ContainElement(heartbeat(1, InstanceStateCrashed)))
			Ω(app().InstanceHeartbeatsAtIndex(2)).Should(HaveLen(2))
			Ω(app().InstanceHeartbeatsAtIndex(2)).Should(ContainElement(heartbeat(2, InstanceStateStarting)))
			Ω(app().InstanceHeartbeatsAtIndex(2)).Should(ContainElement(heartbeat(2, InstanceStateRunning)))
		})
	})

})
