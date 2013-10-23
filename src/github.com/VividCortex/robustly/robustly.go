// Package robustly provides code to handle (and create) infrequent panics.
package robustly

// Copyright (c) 2013 VividCortex, Inc. All rights reserved.
// Please see the LICENSE file for applicable license terms.

import (
	"fmt"
	"github.com/VividCortex/ewma"
	"os"
	"runtime/debug"
	"time"
)

// Run runs the given function robustly, catching and restarting on panics.
// The optional options are a rate limit in crashes per second, and a timeout.
// If the function panics more often than the rate limit, for longer than the
// timeout, then Run aborts and re-throws the panic. A third option controls
// whether to print the stack trace for panics that are intercepted.
func Run(function func(), options ...float64) int {
	rateLimit, timeout := 1.0, 1.0 // TODO

	// We use a moving average to compute the rate of errors per second.
	avg := ewma.NewMovingAverage(timeout)
	before := time.Now()
	var startAboveLimit time.Time
	var belowLimit bool = true
	var beforeTimeout = true
	var totalPanics = 0
	var oktorun bool = true

	for oktorun {
		func() {
			defer func() {
				localErr := recover()
				if localErr == nil {
					oktorun = false // The call to f() exited normally.
					return
				}

				totalPanics++
				after := time.Now()
				duration := after.Sub(before).Seconds()
				if duration > 0 {
					rate := 1.0 / duration
					avg.Add(rate)

					// Figure out whether we're above the rate limit and for how long
					if avg.Value() > rateLimit {
						if belowLimit {
							startAboveLimit = after
						}
						beforeTimeout =
							after.Before(startAboveLimit.Add(time.Second * time.Duration(timeout)))
						belowLimit = false
					} else {
						belowLimit = true
					}
				}
				before = after

				if !belowLimit && !beforeTimeout {
					panic(fmt.Sprintf("giving up after %d errors at %.2f/sec since %s",
						totalPanics, avg.Value(), startAboveLimit))
				}

				if len(options) > 2 && options[2] > 0 {
					fmt.Fprintf(os.Stdout, "%v\n%s\n", localErr, debug.Stack())
				}
			}()
			function()
			return
		}()

	}
	return totalPanics
}
