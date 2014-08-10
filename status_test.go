package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_status(t *testing.T) {
	var status *Status

	Convey("Given a status", t, func() {

		status = &Status{}

		Convey("When started equals expected equals current", func() {
			status.expected = "started"
			status.current = "started"

			Convey("The computed status should be started if alive", func() {
				status.alive = "1"
				So(status.compute(), ShouldEqual, STARTED_STATUS)
			})

			Convey("The computed status should be error if not alive", func() {
				status.alive = ""
				So(status.compute(), ShouldEqual, ERROR_STATUS)
			})

		})

		Convey("When started is expected and current is starting", func() {
			status.expected = "started"
			status.current = "starting"

			Convey("Then computed status should be starting", func() {
				So(status.compute(), ShouldEqual, STARTING_STATUS)

			})

		})

		Convey("When stopped is expected and current is stopped", func() {
			status.expected = "stopped"
			status.current = "stopped"
			Convey("Then computed status should be stopped", func() {

				So(status.compute(), ShouldEqual, STOPPED_STATUS)

			})

		})

		Convey("When stopped is expected and current is stopping", func() {
			status.expected = "stopped"
			status.current = "stopping"

			Convey("Then computed status should be starting", func() {

				So(status.compute(), ShouldEqual, STOPPED_STATUS)

			})

		})

		Convey("When current is passivated", func() {
			status.expected = "passivated"
			status.current = "stopped"

			Convey("Then computed status should be passivated", func() {

				So(status.compute(), ShouldEqual, PASSIVATED_STATUS)

			})

		})

		Convey("When status is nil", func() {
			var status *Status
			status = nil

			Convey("Then the computed status should be started", func() {
				So(status.compute(), ShouldEqual, STARTED_STATUS)
			})

		})

	})

}
