package main

import (
	"testing"
)

func TestParseParameters(t *testing.T) {
	act, _ := parseParameters("w_400,h_300")
	exp := Params{400, 300, DefaultScale, DefaultCroppingMode, DefaultGravity, DefaultFilter}
	if act != exp {
		t.Errorf("Expected: %v, actual: %v", exp, act)
	}

	act, _ = parseParameters("w_200,h_300,c_k,g_c")
	exp = Params{200, 300, DefaultScale, CroppingModeKeepScale, GravityCenter, DefaultFilter}
	if act != exp {
		t.Errorf("Expected: %v, actual: %v", exp, act)
	}
}
