package metricsaccountant_test

import (
	"errors"
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Metrics Accountant", func() {
	var store storepackage.Store
	var accountant MetricsAccountant
	var fakeStoreAdapter *fakestoreadapter.FakeStoreAdapter

	conf, _ := config.DefaultConfig()

	BeforeEach(func() {
		fakeStoreAdapter = fakestoreadapter.New()
		store = storepackage.NewStore(conf, fakeStoreAdapter, fakelogger.NewFakeLogger())
		accountant = New(store)
	})

	Describe("Getting Metrics", func() {
		Context("when the store is empty", func() {
			It("should return a map of 0s", func() {
				metrics, err := accountant.GetMetrics()
				Ω(err).ShouldNot(HaveOccured())
				Ω(metrics).Should(Equal(map[string]float64{
					"StartCrashed":                            0,
					"StartMissing":                            0,
					"StartEvacuating":                         0,
					"StopExtra":                               0,
					"StopDuplicate":                           0,
					"StopEvacuationComplete":                  0,
					"DesiredStateSyncTimeInMilliseconds":      0,
					"ActualStateListenerStoreUsagePercentage": 0,
					"ReceivedHeartbeats":                      0,
					"SavedHeartbeats":                         0,
				}))
			})
		})

		Context("when the store errors for some other reason", func() {
			BeforeEach(func() {
				fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("metrics", errors.New("oops"))
			})

			It("should return an error and an empty map", func() {
				metrics, err := accountant.GetMetrics()
				Ω(err).Should(Equal(errors.New("oops")))
				Ω(metrics).Should(BeEmpty())
			})
		})
	})

	Describe("TrackReceivedHeartbeats", func() {
		It("should record the number of received heartbeats appropriately", func() {
			err := accountant.TrackReceivedHeartbeats(127)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["ReceivedHeartbeats"]).Should(BeNumerically("==", 127))
		})
	})

	Describe("TrackSavedHeartbeats", func() {
		It("should record the number of received heartbeats appropriately", func() {
			err := accountant.TrackSavedHeartbeats(91)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["SavedHeartbeats"]).Should(BeNumerically("==", 91))
		})
	})

	Describe("TrackDesiredStateSyncTime", func() {
		It("should record the passed in time duration appropriately", func() {
			err := accountant.TrackDesiredStateSyncTime(1138 * time.Millisecond)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["DesiredStateSyncTimeInMilliseconds"]).Should(BeNumerically("==", 1138))
		})
	})

	Describe("TrackActualStateListenerStoreUsageFraction", func() {
		It("should record the passed in time duration appropriately", func() {
			err := accountant.TrackActualStateListenerStoreUsageFraction(0.723)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["ActualStateListenerStoreUsagePercentage"]).Should(BeNumerically("==", 72.3))
		})
	})

	Describe("TrackDesiredStateSyncTime", func() {
		It("should record the passed in time duration appropriately", func() {
			err := accountant.TrackDesiredStateSyncTime(1138 * time.Millisecond)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["DesiredStateSyncTimeInMilliseconds"]).Should(BeNumerically("==", 1138))
		})
	})

	Describe("TrackActualStateListenerStoreUsageFraction", func() {
		It("should record the passed in time duration appropriately", func() {
			err := accountant.TrackActualStateListenerStoreUsageFraction(0.723)
			Ω(err).ShouldNot(HaveOccured())
			metrics, err := accountant.GetMetrics()
			Ω(err).ShouldNot(HaveOccured())
			Ω(metrics["ActualStateListenerStoreUsagePercentage"]).Should(BeNumerically("==", 72.3))
		})
	})

	Describe("IncrementSentMessageMetrics", func() {
		var starts []models.PendingStartMessage
		var stops []models.PendingStopMessage
		BeforeEach(func() {
			starts = []models.PendingStartMessage{
				{StartReason: models.PendingStartMessageReasonCrashed},
				{StartReason: models.PendingStartMessageReasonMissing},
				{StartReason: models.PendingStartMessageReasonMissing},
				{StartReason: models.PendingStartMessageReasonEvacuating},
				{StartReason: models.PendingStartMessageReasonEvacuating},
				{StartReason: models.PendingStartMessageReasonEvacuating},
			}

			stops = []models.PendingStopMessage{
				{StopReason: models.PendingStopMessageReasonExtra},
				{StopReason: models.PendingStopMessageReasonDuplicate},
				{StopReason: models.PendingStopMessageReasonDuplicate},
				{StopReason: models.PendingStopMessageReasonEvacuationComplete},
				{StopReason: models.PendingStopMessageReasonEvacuationComplete},
				{StopReason: models.PendingStopMessageReasonEvacuationComplete},
			}
		})

		Context("when the store is empty", func() {
			BeforeEach(func() {
				err := accountant.IncrementSentMessageMetrics(starts, stops)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should increment the metrics and return them when GettingMetrics", func() {
				metrics, err := accountant.GetMetrics()
				Ω(err).ShouldNot(HaveOccured())
				Ω(metrics["StartCrashed"]).Should(BeNumerically("==", 1))
				Ω(metrics["StartMissing"]).Should(BeNumerically("==", 2))
				Ω(metrics["StartEvacuating"]).Should(BeNumerically("==", 3))
				Ω(metrics["StopExtra"]).Should(BeNumerically("==", 1))
				Ω(metrics["StopDuplicate"]).Should(BeNumerically("==", 2))
				Ω(metrics["StopEvacuationComplete"]).Should(BeNumerically("==", 3))
			})
		})

		Context("when the metric already exists", func() {
			BeforeEach(func() {
				err := accountant.IncrementSentMessageMetrics(starts, stops)
				Ω(err).ShouldNot(HaveOccured())
				err = accountant.IncrementSentMessageMetrics(starts, stops)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should increment the metrics and return them when GettingMetrics", func() {
				metrics, err := accountant.GetMetrics()
				Ω(err).ShouldNot(HaveOccured())
				Ω(metrics["StartCrashed"]).Should(BeNumerically("==", 2))
				Ω(metrics["StartMissing"]).Should(BeNumerically("==", 4))
				Ω(metrics["StartEvacuating"]).Should(BeNumerically("==", 6))
				Ω(metrics["StopExtra"]).Should(BeNumerically("==", 2))
				Ω(metrics["StopDuplicate"]).Should(BeNumerically("==", 4))
				Ω(metrics["StopEvacuationComplete"]).Should(BeNumerically("==", 6))
			})
		})

		Context("when the store times out while getting metrics", func() {
			BeforeEach(func() {
				fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("metrics", errors.New("oops"))
			})

			It("should return an error", func() {
				err := accountant.IncrementSentMessageMetrics(starts, stops)
				Ω(err).Should(Equal(errors.New("oops")))
			})
		})

		Context("when the store times out while saving metrics", func() {
			BeforeEach(func() {
				fakeStoreAdapter.SetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("metrics", errors.New("oops"))
			})

			It("should return an error", func() {
				err := accountant.IncrementSentMessageMetrics(starts, stops)
				Ω(err).Should(Equal(errors.New("oops")))
			})
		})
	})
})
