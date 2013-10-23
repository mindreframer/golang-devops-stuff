package robustly

// Copyright (c) 2013 VividCortex, Inc. All rights reserved.
// Please see the LICENSE file for applicable license terms.

import (
	"testing"
	"time"
)

// Panic for a certain number of iterations at a specific time duration.
// Be careful about changing this; the CrashSetup refers specifically to the
// line of code below where Crash() is called.
func panicRateIters(rate time.Duration, iters int, count *int) {
	time.Sleep(rate)
	*count = *count + 1
	if *count <= iters {
		Crash()
	}
}

func TestRobustly1(t *testing.T) {
	CrashSetup("robustly_test.go:18:1")
	tries := 0

	panics := Run(func() { panicRateIters(time.Second, 5, &tries) }, 1, 1)
	if panics != 5 {
		t.Errorf("function panicked %d times, expected 5", panics)
	}
}

func TestRobustly2(t *testing.T) { // just to be sure that crash site printout works
	err := CrashSetup("robustly_test.go:0:0,VERBOSE")
	if err != nil {
		t.Error(err)
	}
	Crash()
}

func TestRobustly3(t *testing.T) {
	CrashSetup("robustly_test.go:18:1")
	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	}()
	tries := 0
	Run(func() { panicRateIters(time.Millisecond*300, 500, &tries) }, 1, 1)
	t.Errorf("this code shouldn't run at all, the defer() should run")
}

func TestRobustly4(t *testing.T) {
	CrashSetup("robustly_test.go:18:1")
	defer func() {
		err := recover()
		if err != nil {
			t.Errorf("got error %v, expected nil", err)
		}
	}()
	tries := 0
	panics := Run(func() { panicRateIters(time.Millisecond*300, 2, &tries) }, 1, 1)
	if panics != 2 {
		t.Errorf("got %d panics, expected 2", panics)
	}
}
