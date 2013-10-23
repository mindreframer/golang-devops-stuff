package storeadapter_test

import (
	. "github.com/cloudfoundry/hm9000/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ETCD Store Adapter", func() {
	var (
		adapter       StoreAdapter
		breakfastNode StoreNode
		lunchNode     StoreNode
	)

	BeforeEach(func() {
		breakfastNode = StoreNode{
			Key:   "/menu/breakfast",
			Value: []byte("waffles"),
		}

		lunchNode = StoreNode{
			Key:   "/menu/lunch",
			Value: []byte("burgers"),
		}

		adapter = NewETCDStoreAdapter(etcdRunner.NodeURLS(), 100)
		err := adapter.Connect()
		Ω(err).ShouldNot(HaveOccured())
	})

	AfterEach(func() {
		adapter.Disconnect()
	})

	Describe("Get", func() {
		BeforeEach(func() {
			err := adapter.Set([]StoreNode{breakfastNode, lunchNode})
			Ω(err).ShouldNot(HaveOccured())
		})

		Context("when getting a key", func() {
			It("should return the appropriate store breakfastNode", func() {
				value, err := adapter.Get("/menu/breakfast")
				Ω(err).ShouldNot(HaveOccured())
				Ω(value).Should(Equal(breakfastNode))
			})
		})

		Context("When getting a non-existent key", func() {
			It("should return an error", func() {
				value, err := adapter.Get("/not_a_key")
				Ω(err).Should(Equal(ErrorKeyNotFound))
				Ω(value).Should(BeZero())
			})
		})

		Context("when getting a directory", func() {
			It("should return an error", func() {
				value, err := adapter.Get("/menu")
				Ω(err).Should(Equal(ErrorNodeIsDirectory))
				Ω(value).Should(BeZero())
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				etcdRunner.Stop()
			})

			AfterEach(func() {
				etcdRunner.Start()
			})

			It("should return a timeout error", func() {
				value, err := adapter.Get("/foo/bar")
				Ω(err).Should(Equal(ErrorTimeout))
				Ω(value).Should(BeZero())
			})
		})
	})

	Describe("Set", func() {
		It("should be able to set multiple things to the store at once", func() {
			err := adapter.Set([]StoreNode{breakfastNode, lunchNode})
			Ω(err).ShouldNot(HaveOccured())

			menu, err := adapter.ListRecursively("/menu")
			Ω(err).ShouldNot(HaveOccured())
			Ω(menu.ChildNodes).Should(HaveLen(2))
			Ω(menu.ChildNodes).Should(ContainElement(breakfastNode))
			Ω(menu.ChildNodes).Should(ContainElement(lunchNode))
		})

		Context("Setting to an existing node", func() {
			BeforeEach(func() {
				err := adapter.Set([]StoreNode{breakfastNode, lunchNode})
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should be able to update existing entries", func() {
				lunchNode.Value = []byte("steak")
				err := adapter.Set([]StoreNode{breakfastNode, lunchNode})
				Ω(err).ShouldNot(HaveOccured())

				menu, err := adapter.ListRecursively("/menu")
				Ω(err).ShouldNot(HaveOccured())
				Ω(menu.ChildNodes).Should(HaveLen(2))
				Ω(menu.ChildNodes).Should(ContainElement(breakfastNode))
				Ω(menu.ChildNodes).Should(ContainElement(lunchNode))
			})

			It("should error when attempting to set to a directory", func() {
				dirNode := StoreNode{
					Key:   "/menu",
					Value: []byte("oops!"),
				}

				err := adapter.Set([]StoreNode{dirNode})
				Ω(err).Should(Equal(ErrorNodeIsDirectory))
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				etcdRunner.Stop()
			})

			AfterEach(func() {
				etcdRunner.Start()
			})

			It("should return a timeout error", func() {
				err := adapter.Set([]StoreNode{breakfastNode})
				Ω(err).Should(Equal(ErrorTimeout))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			err := adapter.Set([]StoreNode{breakfastNode, lunchNode})
			Ω(err).ShouldNot(HaveOccured())
		})

		Context("When listing a directory", func() {
			It("Should list directory contents", func() {
				value, err := adapter.ListRecursively("/menu")
				Ω(err).ShouldNot(HaveOccured())
				Ω(value.Key).Should(Equal("/menu"))
				Ω(value.Dir).Should(BeTrue())
				Ω(value.ChildNodes).Should(HaveLen(2))
				Ω(value.ChildNodes).Should(ContainElement(breakfastNode))
				Ω(value.ChildNodes).Should(ContainElement(lunchNode))
			})
		})

		Context("when listing a directory that contains directories", func() {
			var (
				firstCourseDinnerNode  StoreNode
				secondCourseDinnerNode StoreNode
			)

			BeforeEach(func() {
				firstCourseDinnerNode = StoreNode{
					Key:   "/menu/dinner/first_course",
					Value: []byte("Salad"),
				}
				secondCourseDinnerNode = StoreNode{
					Key:   "/menu/dinner/second_course",
					Value: []byte("Brisket"),
				}
				err := adapter.Set([]StoreNode{firstCourseDinnerNode, secondCourseDinnerNode})

				Ω(err).ShouldNot(HaveOccured())
			})

			Context("when listing the root directory", func() {
				It("should list the contents recursively", func() {
					value, err := adapter.ListRecursively("/")
					Ω(err).ShouldNot(HaveOccured())
					Ω(value.Key).Should(Equal("/"))
					Ω(value.Dir).Should(BeTrue())
					Ω(value.ChildNodes).Should(HaveLen(1))
					menuNode := value.ChildNodes[0]
					Ω(menuNode.Key).Should(Equal("/menu"))
					Ω(menuNode.Value).Should(BeEmpty())
					Ω(menuNode.Dir).Should(BeTrue())
					Ω(menuNode.ChildNodes).Should(HaveLen(3))
					Ω(menuNode.ChildNodes).Should(ContainElement(breakfastNode))
					Ω(menuNode.ChildNodes).Should(ContainElement(lunchNode))

					var dinnerNode StoreNode
					for _, node := range menuNode.ChildNodes {
						if node.Key == "/menu/dinner" {
							dinnerNode = node
							break
						}
					}
					Ω(dinnerNode.Dir).Should(BeTrue())
					Ω(dinnerNode.ChildNodes).Should(ContainElement(firstCourseDinnerNode))
					Ω(dinnerNode.ChildNodes).Should(ContainElement(secondCourseDinnerNode))
				})
			})

			Context("when listing another directory", func() {
				It("should list the contents recursively", func() {
					menuNode, err := adapter.ListRecursively("/menu")
					Ω(err).ShouldNot(HaveOccured())
					Ω(menuNode.Key).Should(Equal("/menu"))
					Ω(menuNode.Value).Should(BeEmpty())
					Ω(menuNode.Dir).Should(BeTrue())
					Ω(menuNode.ChildNodes).Should(HaveLen(3))
					Ω(menuNode.ChildNodes).Should(ContainElement(breakfastNode))
					Ω(menuNode.ChildNodes).Should(ContainElement(lunchNode))

					var dinnerNode StoreNode
					for _, node := range menuNode.ChildNodes {
						if node.Key == "/menu/dinner" {
							dinnerNode = node
							break
						}
					}
					Ω(dinnerNode.Dir).Should(BeTrue())
					Ω(dinnerNode.ChildNodes).Should(ContainElement(firstCourseDinnerNode))
					Ω(dinnerNode.ChildNodes).Should(ContainElement(secondCourseDinnerNode))
				})
			})
		})

		Context("when listing an empty directory", func() {
			It("should return an empty list of breakfastNodes, and not error", func() {
				err := adapter.Set([]StoreNode{
					StoreNode{
						Key:   "/empty_dir/temp",
						Value: []byte("foo"),
					},
				})
				Ω(err).ShouldNot(HaveOccured())

				err = adapter.Delete("/empty_dir/temp")
				Ω(err).ShouldNot(HaveOccured())

				value, err := adapter.ListRecursively("/empty_dir")
				Ω(err).ShouldNot(HaveOccured())
				Ω(value.Key).Should(Equal("/empty_dir"))
				Ω(value.Dir).Should(BeTrue())
				Ω(value.ChildNodes).Should(HaveLen(0))
			})
		})

		Context("when listing a non-existent key", func() {
			It("should return an error", func() {
				value, err := adapter.ListRecursively("/nothing-here")
				Ω(err).Should(Equal(ErrorKeyNotFound))
				Ω(value).Should(BeZero())
			})
		})

		Context("When listing an entry", func() {
			It("should return an error", func() {
				value, err := adapter.ListRecursively("/menu/breakfast")
				Ω(err).Should(HaveOccured())
				Ω(err).Should(Equal(ErrorNodeIsNotDirectory))
				Ω(value).Should(BeZero())
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				etcdRunner.Stop()
			})

			AfterEach(func() {
				etcdRunner.Start()
			})

			It("should return a timeout error", func() {
				value, err := adapter.ListRecursively("/menu")
				Ω(err).Should(Equal(ErrorTimeout))
				Ω(value).Should(BeZero())
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			err := adapter.Set([]StoreNode{breakfastNode})
			Ω(err).ShouldNot(HaveOccured())
		})

		Context("when deleting an existing key", func() {
			It("should delete the key", func() {
				err := adapter.Delete("/menu/breakfast")
				Ω(err).ShouldNot(HaveOccured())

				value, err := adapter.Get("/menu/breakfat")
				Ω(err).Should(Equal(ErrorKeyNotFound))
				Ω(value).Should(BeZero())
			})
		})

		Context("when deleting a non-existing key", func() {
			It("should error", func() {
				err := adapter.Delete("/not-a-key")
				Ω(err).Should(Equal(ErrorKeyNotFound))
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				etcdRunner.Stop()
			})

			AfterEach(func() {
				etcdRunner.Start()
			})

			It("should return a timeout error", func() {
				err := adapter.Delete("/menu/breakfast")
				Ω(err).Should(Equal(ErrorTimeout))
			})
		})
	})

	Context("When setting a key with a non-zero TTL", func() {
		It("should stay in the adapter for its TTL and then disappear", func() {
			breakfastNode.TTL = 1
			err := adapter.Set([]StoreNode{breakfastNode})
			Ω(err).ShouldNot(HaveOccured())

			_, err = adapter.Get("/menu/breakfast")
			Ω(err).ShouldNot(HaveOccured())

			Eventually(func() interface{} {
				_, err = adapter.Get("/menu/breakfast")
				return err
			}, 1.05, 0.01).Should(Equal(ErrorKeyNotFound))
		})
	})
})
