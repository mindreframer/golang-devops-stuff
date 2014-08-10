package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_cluster(t *testing.T) {
	var cluster *ServiceCluster

	Convey("Given an service cluster", t, func() {
		cluster = &ServiceCluster{}

		Convey("When the cluster is initialized", func() {
			Convey("Then it should be empty", func() {
				So(len(cluster.instances), ShouldEqual, 0)

			})
		})

		Convey("When the cluster contains an inactive service", func() {
			cluster.Add(getService("1", "nxio-0001", false))
			Convey("Then it can't get a next service", func() {
				service, err := cluster.Next()

				So(len(cluster.instances), ShouldEqual, 1)
				So(service, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When the cluster contains active service", func() {
			cluster.Add(getService("2", "nxio-0001", true))
			Convey("Then it can get a next service", func() {
				service, err := cluster.Next()

				So(len(cluster.instances), ShouldEqual, 1)
				So(service, ShouldNotBeNil)
				So(err, ShouldBeNil)
			})

			Convey("Then returned service should always be the same", func() {
				service, _ := cluster.Next()
				firstKey := service.index
				service, _ = cluster.Next()
				So(service.index, ShouldEqual, firstKey)

			})

		})

		Convey("When the cluster contains several services", func() {
			cluster.Add(getService("1", "nxio-0001", true))
			cluster.Add(getService("2", "nxio-0001", false))
			cluster.Add(getService("3", "nxio-0001", true))

			Convey("Then it should loadbalance between services", func() {
				service, err := cluster.Next()
				So(service, ShouldNotBeNil)
				So(err, ShouldBeNil)

				firstKey := service.index

				service, err = cluster.Next()
				So(service, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(service.index, ShouldNotEqual, firstKey)
			})

			Convey("Then it should never loadbalance on an inactive service", func() {
				for i := 0; i < len(cluster.instances); i++ {
					service, err := cluster.Next()
					So(service, ShouldNotBeNil)
					So(err, ShouldBeNil)
					So(service.index, ShouldNotEqual, "2")
				}
			})

			Convey("Then it can get each service by its key", func() {

				service := cluster.Get("1")
				So(service.index, ShouldEqual, "1")
				So(service.status.current, ShouldEqual, "started")

				service = cluster.Get("2")
				So(service.index, ShouldEqual, "2")
				So(service.status.current, ShouldEqual, "stopped")
			})

		})

		Convey("When removing a key to a cluster", func() {
			cluster.Add(getService("1", "nxio-0001", true))
			cluster.Add(getService("2", "nxio-0001", false))
			cluster.Add(getService("3", "nxio-0001", true))

			initSize := len(cluster.instances)

			cluster.Remove("2")

			Convey("Then it should containe one less instance", func() {
				So(len(cluster.instances), ShouldEqual, initSize-1)

			})
		})

	})

}

func Test_Service(t *testing.T) {
	var service1, service2 *Service

	Convey("Given two service with same values", t, func() {
		service1 = getService("1", "nxio-0001", true)
		service2 = getService("1", "nxio-0001", true)
		Convey("When i dont change anything", func() {
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, true)

			})

		})

		Convey("When host is not the same", func() {
			service2.location.Host = "otherhost"
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, false)

			})

		})

		Convey("When port is not the same", func() {
			service2.location.Port = 9090
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, false)

			})

		})

		Convey("When current status is not the same", func() {
			service2.status.current = "other"
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, false)

			})

		})

		Convey("When expected status is not the same", func() {
			service2.status.expected = "other"
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, false)

			})

		})
		Convey("When alive status is not the same", func() {
			service2.status.alive = "other"
			Convey("Then they are equal", func() {

				So(service1.equals(service2), ShouldEqual, false)

			})

		})
	})
}

func getService(index string, name string, active bool) *Service {
	var s *Status

	if active {
		s = &Status{"1", "started", "started", &Service{}}
	} else {
		s = &Status{"", "stopped", "started", &Service{}}
	}

	return &Service{
		index:      index,
		location: &location{"127.0.0.1", 8080},
		domain:   "dummydomain.com",
		name:     name,
		status:   s}

}
