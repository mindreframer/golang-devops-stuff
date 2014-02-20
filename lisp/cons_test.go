package lisp

import (
	"testing"
)

func cons() Cons {
	v1 := &Value{numberValue, 1.0}
	v2 := &Value{numberValue, 2.0}
	v3 := &Value{numberValue, 3.0}
	c2 := &Value{consValue, &Cons{v3, &Value{nilValue, nil}}}
	c1 := &Value{consValue, &Cons{v2, c2}}
	return Cons{v1, c1}
}

func TestConsMap(t *testing.T) {
	s, _ := cons().Map(func(v Value) (Value, error) {
		return Value{numberValue, v.val.(float64) + 1}, nil
	})
	if len(s) != 3 || s[0].val != 2.0 || s[1].val != 3.0 || s[2].val != 4.0 {
		t.Errorf("Expected (1 2 3), got %v", s)
	}
}

func TestConsLen(t *testing.T) {
	got := cons().Len()
	if got != 3 {
		t.Errorf("Expected 3, got %v\n", got)
	}
}

func TestConsVector(t *testing.T) {
	s := cons().Vector()
	if len(s) != 3 || s[0].val != 1.0 || s[1].val != 2.0 || s[2].val != 3.0 {
		t.Errorf("Expected (1 2 3), got %v", s)
	}
}

func TestConsString(t *testing.T) {
	expected := "(1 2 3)"
	s := cons().String()
	if s != expected {
		t.Errorf("Cons.String() failed. Expected %v, got %v", expected, s)
	}
}
