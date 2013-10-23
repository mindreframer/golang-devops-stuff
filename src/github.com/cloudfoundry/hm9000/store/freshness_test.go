package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"time"
)

var _ = Describe("Freshness", func() {
	var (
		store       Store
		etcdAdapter storeadapter.StoreAdapter
		conf        config.Config
	)

	conf, _ = config.DefaultConfig()

	BeforeEach(func() {
		etcdAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), conf.StoreMaxConcurrentRequests)
		err := etcdAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		store = NewStore(conf, etcdAdapter, fakelogger.NewFakeLogger())
	})

	Describe("Bumping freshness", func() {
		bumpingFreshness := func(key string, ttl uint64, bump func(store Store, timestamp time.Time) error) {
			var timestamp time.Time

			BeforeEach(func() {
				timestamp = time.Now()
			})

			Context("when the key is missing", func() {
				BeforeEach(func() {
					_, err := etcdAdapter.Get(key)
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

					err = bump(store, timestamp)
					Ω(err).ShouldNot(HaveOccured())
				})

				It("should create the key with the current timestamp and a TTL", func() {
					value, err := etcdAdapter.Get(key)

					Ω(err).ShouldNot(HaveOccured())

					var freshnessTimestamp models.FreshnessTimestamp
					json.Unmarshal(value.Value, &freshnessTimestamp)

					Ω(freshnessTimestamp.Timestamp).Should(Equal(timestamp.Unix()))
					Ω(value.TTL).Should(BeNumerically("==", ttl-1))
					Ω(value.Key).Should(Equal(key))
				})
			})

			Context("when the key is present", func() {
				BeforeEach(func() {
					err := bump(store, time.Unix(100, 0))
					Ω(err).ShouldNot(HaveOccured())
					err = bump(store, timestamp)
					Ω(err).ShouldNot(HaveOccured())
				})

				It("should bump the key's TTL but not change the timestamp", func() {
					value, err := etcdAdapter.Get(key)

					Ω(err).ShouldNot(HaveOccured())

					Ω(value.TTL).Should(BeNumerically("==", ttl-1))

					var freshnessTimestamp models.FreshnessTimestamp
					json.Unmarshal(value.Value, &freshnessTimestamp)

					Ω(freshnessTimestamp.Timestamp).Should(BeNumerically("==", 100))
					Ω(value.Key).Should(Equal(key))
				})
			})
		}

		Context("the actual state", func() {
			bumpingFreshness(conf.ActualFreshnessKey, conf.ActualFreshnessTTL(), Store.BumpActualFreshness)
		})

		Context("the desired state", func() {
			bumpingFreshness(conf.DesiredFreshnessKey, conf.DesiredFreshnessTTL(), Store.BumpDesiredFreshness)
		})
	})

	Describe("Verifying the store's freshness", func() {
		Context("when neither the desired or actual state is fresh", func() {
			It("should return the appropriate error", func() {
				err := store.VerifyFreshness(time.Unix(100, 0))
				Ω(err).Should(Equal(ActualAndDesiredAreNotFreshError))
			})
		})

		Context("when only the desired state is not fresh", func() {
			It("should return the appropriate error", func() {
				store.BumpActualFreshness(time.Unix(100, 0))
				err := store.VerifyFreshness(time.Unix(int64(100+conf.ActualFreshnessTTL()), 0))
				Ω(err).Should(Equal(DesiredIsNotFreshError))
			})
		})

		Context("when only the actual state is not fresh", func() {
			It("should return the appropriate error", func() {
				store.BumpDesiredFreshness(time.Unix(100, 0))
				err := store.VerifyFreshness(time.Unix(100, 0))
				Ω(err).Should(Equal(ActualIsNotFreshError))
			})
		})

		Context("when both are fresh", func() {
			It("should not error", func() {
				store.BumpActualFreshness(time.Unix(100, 0))
				store.BumpDesiredFreshness(time.Unix(100, 0))
				err := store.VerifyFreshness(time.Unix(int64(100+conf.ActualFreshnessTTL()), 0))
				Ω(err).ShouldNot(HaveOccured())
			})
		})
	})

	Describe("Checking desired state freshness", func() {
		Context("if the freshness key is not present", func() {
			It("returns that the state is not fresh", func() {
				fresh, err := store.IsDesiredStateFresh()
				Ω(err).ShouldNot(HaveOccured())
				Ω(fresh).Should(BeFalse())
			})
		})

		Context("if the freshness key is present", func() {
			BeforeEach(func() {
				store.BumpDesiredFreshness(time.Unix(100, 0))
			})

			It("returns that the state is fresh", func() {
				fresh, err := store.IsDesiredStateFresh()
				Ω(err).ShouldNot(HaveOccured())
				Ω(fresh).Should(BeTrue())
			})
		})

		Context("when the store returns an error", func() {
			BeforeEach(func() {
				err := etcdAdapter.Set([]storeadapter.StoreNode{
					storeadapter.StoreNode{
						Key:   "/desired-fresh/mwahaha",
						Value: []byte("i'm a directory...."),
					},
				})
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the store's error", func() {
				fresh, err := store.IsDesiredStateFresh()
				Ω(err).Should(Equal(storeadapter.ErrorNodeIsDirectory))
				Ω(fresh).Should(BeFalse())
			})
		})
	})

	Describe("Checking actual state freshness", func() {
		Context("if the freshness key is not present", func() {
			It("returns that the state is not fresh", func() {
				fresh, err := store.IsActualStateFresh(time.Unix(130, 0))
				Ω(err).ShouldNot(HaveOccured())
				Ω(fresh).Should(BeFalse())
			})
		})

		Context("if the freshness key is present", func() {
			BeforeEach(func() {
				store.BumpActualFreshness(time.Unix(100, 0))
			})

			Context("if the creation time of the key is outside the last x seconds", func() {
				It("returns that the state is fresh", func() {
					fresh, err := store.IsActualStateFresh(time.Unix(130, 0))
					Ω(err).ShouldNot(HaveOccured())
					Ω(fresh).Should(BeTrue())
				})
			})

			Context("if the creation time of the key is within the last x seconds", func() {
				It("returns that the state is not fresh", func() {
					fresh, err := store.IsActualStateFresh(time.Unix(129, 0))
					Ω(err).ShouldNot(HaveOccured())
					Ω(fresh).Should(BeFalse())
				})
			})

			Context("if the freshness key fails to parse", func() {
				BeforeEach(func() {
					etcdAdapter.Set([]storeadapter.StoreNode{
						storeadapter.StoreNode{
							Key:   "/actual-fresh",
							Value: []byte("ß"),
						},
					})
				})

				It("should return an error", func() {
					fresh, err := store.IsActualStateFresh(time.Unix(129, 0))
					Ω(err).Should(HaveOccured())
					Ω(fresh).Should(BeFalse())
				})
			})
		})

		Context("when the store returns an error", func() {
			BeforeEach(func() {
				err := etcdAdapter.Set([]storeadapter.StoreNode{
					storeadapter.StoreNode{
						Key:   "/actual-fresh/mwahaha",
						Value: []byte("i'm a directory...."),
					},
				})
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the store's error", func() {
				fresh, err := store.IsActualStateFresh(time.Unix(130, 0))
				Ω(err).Should(Equal(storeadapter.ErrorNodeIsDirectory))
				Ω(fresh).Should(BeFalse())
			})
		})
	})
})
