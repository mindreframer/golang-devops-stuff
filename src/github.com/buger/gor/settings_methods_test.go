package main

import (
	"testing"
)

func TestHTTPMethods(t *testing.T) {
	methods := HTTPMethods{}

	methods.Set("lower")
	methods.Set("UPPER")

	if !methods.Contains("LOWER") {
		t.Error("Does not contain LOWER")
	}

	if !methods.Contains("UPPER") {
		t.Error("Does not contain UPPER")
	}

	if methods.Contains("ABSENT") {
		t.Error("Does contain ABSENT")
	}
}
