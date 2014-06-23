package main

import (
	"image"
	"testing"
)

func TestCalculateTopLeftPointFromGravity(t *testing.T) {
	exp := image.Point{200, 0}
	act := calculateTopLeftPointFromGravity(GravityNorth, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("N failed", act, exp)
	}

	exp = image.Point{400, 0}
	act = calculateTopLeftPointFromGravity(GravityNorthEast, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("NE failed", act, exp)
	}

	exp = image.Point{400, 150}
	act = calculateTopLeftPointFromGravity(GravityEast, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("E failed", act, exp)
	}

	exp = image.Point{400, 300}
	act = calculateTopLeftPointFromGravity(GravitySouthEast, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("SE failed", act, exp)
	}
	exp = image.Point{200, 300}
	act = calculateTopLeftPointFromGravity(GravitySouth, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("S failed", act, exp)
	}
	exp = image.Point{0, 300}
	act = calculateTopLeftPointFromGravity(GravitySouthWest, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("SW failed", act, exp)
	}
	exp = image.Point{0, 150}
	act = calculateTopLeftPointFromGravity(GravityWest, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("W failed", act, exp)
	}

	exp = image.Point{0, 0}
	act = calculateTopLeftPointFromGravity(GravityNorthWest, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("NW failed", act, exp)
	}

	exp = image.Point{200, 150}
	act = calculateTopLeftPointFromGravity(GravityCenter, 400, 300, 800, 600)
	if act != exp {
		t.Errorf("C failed", act, exp)
	}
}
