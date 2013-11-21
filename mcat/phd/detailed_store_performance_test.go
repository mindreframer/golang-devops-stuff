package phd

import (
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/storerunner"

	"fmt"
	"math/rand"
	"time"
)

var numRecords = 512
var storeTypes = []string{"ETCD", "Zookeeper"}
var nodeCounts = []int{1, 3, 5, 7}
var concurrencies = []int{1, 5, 10, 15, 20, 25, 30}
var recordSizes = []int{128, 256, 512, 1024, 2048, 4096}

var _ = Describe("Detailed Store Performance", func() {
	for _, storeType := range storeTypes {
		storeType := storeType
		for _, nodes := range nodeCounts {
			nodes := nodes
			for _, concurrency := range concurrencies {
				concurrency := concurrency
				Context(fmt.Sprintf("With %d %s nodes (%d concurrent requests at a time)", nodes, storeType, concurrency), func() {
					var storeAdapter storeadapter.StoreAdapter

					BeforeEach(func() {
						if storeType == "ETCD" {
							storeRunner = storerunner.NewETCDClusterRunner(5001, nodes)
							storeRunner.Start()

							storeAdapter = storeadapter.NewETCDStoreAdapter(storeRunner.NodeURLS(), workerpool.NewWorkerPool(concurrency))
							err := storeAdapter.Connect()
							Ω(err).ShouldNot(HaveOccured())
						} else if storeType == "Zookeeper" {
							storeRunner = storerunner.NewZookeeperClusterRunner(2181, nodes)
							storeRunner.Start()

							storeAdapter = storeadapter.NewZookeeperStoreAdapter(storeRunner.NodeURLS(), workerpool.NewWorkerPool(concurrency), &timeprovider.RealTimeProvider{}, time.Second)
							err := storeAdapter.Connect()
							Ω(err).ShouldNot(HaveOccured())
						}
					})

					AfterEach(func() {
						storeAdapter.Disconnect()
						storeRunner.Stop()
						storeRunner = nil
					})

					randomBytes := func(sizeInBytes int) []byte {
						seedBytes := []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'}
						randomBytes := make([]byte, sizeInBytes)
						for i := 0; i < sizeInBytes; i++ {
							randomBytes[i] = seedBytes[rand.Intn(len(seedBytes))]
						}

						return randomBytes
					}

					for _, recordSize := range recordSizes {
						recordSize := recordSize

						Measure(fmt.Sprintf("Read/Write Performance With record size: %dbytes (will generate %d records)", recordSize, numRecords), func(b Benchmarker) {
							data := make([]storeadapter.StoreNode, numRecords)
							for i := 0; i < numRecords; i++ {
								data[i] = storeadapter.StoreNode{
									Key:   fmt.Sprintf("/record/%d", i),
									Value: randomBytes(recordSize),
									TTL:   0,
								}
							}

							b.Time("writing to the store", func() {
								err := storeAdapter.Set(data)
								Ω(err).ShouldNot(HaveOccured())
							}, StorePerformanceReport{
								Subject:       "write",
								StoreType:     storeType,
								NumStoreNodes: nodes,
								RecordSize:    recordSize,
								NumRecords:    numRecords,
								Concurrency:   concurrency,
							})

							b.Time("reading from the store", func() {
								node, err := storeAdapter.ListRecursively("/record")
								Ω(err).ShouldNot(HaveOccured())
								Ω(len(node.ChildNodes)).Should(Equal(numRecords), "Didn't find the correct number of entries in the store")
							}, StorePerformanceReport{
								Subject:       "read",
								StoreType:     storeType,
								NumStoreNodes: nodes,
								RecordSize:    recordSize,
								NumRecords:    numRecords,
								Concurrency:   concurrency,
							})
						}, 5)
					}
				})
			}
		}
	}
})
