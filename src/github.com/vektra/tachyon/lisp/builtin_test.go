package lisp

import "testing"

func num(i float64) Value {
	return Value{numberValue, i}
}

func TestCar(t *testing.T) {
	a, b := Value{stringValue, "a"}, Value{stringValue, "b"}
	cons := Value{consValue, &Cons{&a, &b}}
	if response, err := builtin.Car(cons); response != a || err != nil {
		t.Errorf("Car %v should be %v, was %v", cons, a, response)
	}
}

func TestCdr(t *testing.T) {
	a, b := Value{stringValue, "a"}, Value{stringValue, "b"}
	cons := Value{consValue, &Cons{&a, &b}}
	if response, err := builtin.Cdr(cons); response != b || err != nil {
		t.Errorf("Cdr %v should be %v, was %v", cons, b, response)
	}
}

func TestAdd(t *testing.T) {
	cons := Cons{&Value{symbolValue, "+"}, nil}
	if !cons.isBuiltin() {
		t.Errorf("+ is not correcly setup")
	}

	if sum, err := builtin.Add(num(1), num(2), num(3)); sum != num(6) || err != nil {
		t.Errorf("1 + 2 + 3 should == 6, is %v, error: %v", sum, err)
	}
}

func TestSub(t *testing.T) {
	if sum, err := builtin.Sub(num(5), num(2), num(1)); sum != num(2) || err != nil {
		t.Errorf("5 - 2 - 1 should == 2, is %v, error: %v", sum, err)
	}
}

func TestMul(t *testing.T) {
	if sum, err := builtin.Mul(num(2), num(3), num(4)); sum != num(24) || err != nil {
		t.Errorf("2 * 3 * 4 should == 24, is %v, error: %v", sum, err)
	}
}

func TestGt(t *testing.T) {
	if result, err := builtin.Gt(num(4), num(3), num(2)); result == False || err != nil {
		t.Errorf("4 > 3 > 2 should == true, is %v, error: %v", result, err)
	}

	if result, err := builtin.Gt(num(4), num(4), num(2)); result == True || err != nil {
		t.Errorf("4 > 4 > 2 should == true, is %v, error: %v", result, err)
	}
}

func TestLt(t *testing.T) {
	if result, err := builtin.Lt(num(2), num(3), num(4)); result == False || err != nil {
		t.Errorf("2 < 3 < 4 should == true, is %v, error: %v", result, err)
	}
}

func TestGte(t *testing.T) {
	if result, err := builtin.Gte(num(4), num(4), num(2)); result == False || err != nil {
		t.Errorf("4 >= 4 >= 2 should == true, is %v, error: %v", result, err)
	}
}

func TestLte(t *testing.T) {
	if result, err := builtin.Lte(num(2), num(2), num(4)); result == False || err != nil {
		t.Errorf("2 <= 2 <= 4 should == true, is %v, error: %v", result, err)
	}
}
