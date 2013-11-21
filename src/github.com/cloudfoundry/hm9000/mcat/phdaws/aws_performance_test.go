package phd_aws

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/storecassandra"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"tux21b.org/v1/gocql"

	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"

	"fmt"
)

var numberOfApps = []int{30, 100, 300, 1000, 3000, 10000}
var numberOfInstancesPerApp = 2

var justOnce = false
var _ = Describe("Benchmarking AWS MCAT ", func() {
	var store *storecassandra.StoreCassandra
	BeforeEach(func() {
		if !justOnce {
			conf, _ := config.DefaultConfig()
			var err error
			store, err = storecassandra.New([]string{"127.0.0.1:9042"}, gocql.One, conf, timeprovider.NewTimeProvider())
			Ω(err).ShouldNot(HaveOccured())
			justOnce = true
		}
	})

	for _, numApps := range numberOfApps {
		numApps := numApps
		iteration := 1
		Context(fmt.Sprintf("With %d apps", numApps), func() {
			Measure("Read/Write/Delete Performance", func(b Benchmarker) {
				fmt.Printf("%d apps iteration %d\n", numApps, iteration)
				iteration += 1
				heartbeat := models.Heartbeat{
					DeaGuid:            models.Guid(),
					InstanceHeartbeats: []models.InstanceHeartbeat{},
				}
				n := 0
				for i := 0; i < numApps; i++ {
					app := appfixture.NewAppFixture()
					for j := 0; j < numberOfInstancesPerApp; j++ {
						heartbeat.InstanceHeartbeats = append(heartbeat.InstanceHeartbeats, app.InstanceAtIndex(j).Heartbeat())
						n += 1
					}
				}

				b.Time("WRITE", func() {
					err := store.SyncHeartbeats(heartbeat)
					Ω(err).ShouldNot(HaveOccured())
				}, StorePerformanceReport{
					NumApps: numApps,
					Subject: "write",
				})

				b.Time("READ", func() {
					nodes, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccured())
					Ω(len(nodes)).Should(Equal(numApps*numberOfInstancesPerApp), "Didn't find the correct number of entries in the store")
				}, StorePerformanceReport{
					NumApps: numApps,
					Subject: "read",
				})

				b.Time("DELETE", func() {
					err := store.TruncateActualState()
					Ω(err).ShouldNot(HaveOccured())
				}, StorePerformanceReport{
					NumApps: numApps,
					Subject: "delete",
				})
			}, 5)
		})
	}
})
