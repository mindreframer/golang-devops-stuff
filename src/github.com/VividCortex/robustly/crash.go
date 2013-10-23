package robustly

// Copyright (c) 2013 VividCortex, Inc. All rights reserved.
// Please see the LICENSE file for applicable license terms.

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// This function pointer is nil initially, but after CrashSetup(), will
// be a pointer to a function that'll cause a panic.
var crash func(...int)
var verbose bool

// Crash causes a crash if you configured the callsite to crash with
// CrashSetup.  If you pass it a "calldepth" option, it will examine the stack
// "calldepth" frames up to check whether it is at a crash site.
func Crash(calldepth ...int) {
	if crash != nil {
		crash(calldepth...)
	}
	if verbose {
		depth := 2 // because we're calling from here instead of crash()
		if len(calldepth) > 0 {
			depth = calldepth[0] - 1
		}
		file, line := getCallSite(depth)
		fmt.Printf("crash site at %s:%d\n", file, line)
	}
}

// CrashSetup should be called to configure crash sites in your code. It parses
// and saves a list of crash sites and their probabilities of crashing, and then
// makes the crash() function crash probabilistically when called from one of
// the specified crash sites. An example spec:
//   client.go:53:.003,server.go:18:.02
// That will cause a crash .003 of the time at client.go line 53, and .02 of the time
// at server.go line 18.
func CrashSetup(spec string) error {
	if spec == "" { // Crashing is disabled
		crash = nil
		return nil
	}

	if strings.Contains(spec, "VERBOSE") {
		verbose = true
	}

	// site stores the parsed file/line pairs from the config
	type site struct {
		file string
		line int64
	}

	sites := make(map[site]float64)
	for _, s := range strings.Split(spec, ",") {
		if s != "VERBOSE" {
			file, line, probability, err := newSite(s)
			if err != nil {
				return err
			}
			sites[site{file: file, line: line}] = probability
		}
	}

	// Generate the function that causes crashes.
	crash = func(calldepth ...int) {
		file, line := getCallSite(calldepth...)

		chance := sites[site{
			file: file,
			line: int64(line),
		}]

		if chance > 0 && rand.Float64() <= chance {
			panic(fmt.Sprintf("crash injected at %s:%d, probability %f", file, line, chance))
		}
	}

	return nil
}

// Figure out where the robustly.Crash() was called from
func getCallSite(calldepth ...int) (string, int) {
	depth := 3
	if len(calldepth) > 0 {
		depth = calldepth[0]
	}
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = "???"
		line = 0
	}
	file = filepath.Base(file)
	return file, line
}

// Parse a crash site spec; return values: line, file, probability, error
func newSite(s string) (string, int64, float64, error) {
	parts := strings.Split(s, ":")
	if len(parts) == 3 {
		file := parts[0]
		line, intParseErr := strconv.ParseInt(parts[1], 10, 64)
		if intParseErr == nil {
			prob, floatParseErr := strconv.ParseFloat(parts[2], 64)
			if floatParseErr == nil {
				return file, line, prob, nil
			}
		}
	}
	return "", 0, 0, fmt.Errorf("invalid crash site spec '%s'", s)
}
