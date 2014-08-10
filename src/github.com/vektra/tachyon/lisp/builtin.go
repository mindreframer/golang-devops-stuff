package lisp

import "fmt"

type Builtin struct{}

var builtin = Builtin{}

var builtin_commands = map[string]string{
	"+":       "Add",
	"-":       "Sub",
	"*":       "Mul",
	"==":      "Eq",
	">":       "Gt",
	"<":       "Lt",
	">=":      "Gte",
	"<=":      "Lte",
	"display": "Display",
	"cons":    "Cons",
	"car":     "Car",
	"cdr":     "Cdr",
}

func (Builtin) Display(vars ...Value) (Value, error) {
	if len(vars) == 1 {
		fmt.Println(vars[0])
	} else {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
	return vars[0], nil
}

func (Builtin) Cons(vars ...Value) (Value, error) {
	if len(vars) == 2 {
		cons := Cons{&vars[0], &vars[1]}
		return Value{consValue, &cons}, nil
	} else {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
}

func (Builtin) Car(vars ...Value) (Value, error) {
	if len(vars) == 1 && vars[0].typ == consValue {
		cons := vars[0].Cons()
		return *cons.car, nil
	} else {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
}

func (Builtin) Cdr(vars ...Value) (Value, error) {
	if len(vars) == 1 && vars[0].typ == consValue {
		cons := vars[0].Cons()
		return *cons.cdr, nil
	} else {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
}

func (Builtin) Add(vars ...Value) (Value, error) {
	var sum int64
	for _, v := range vars {
		if v.typ == numberValue {
			sum += v.Number()
		} else {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		}
	}
	return Value{numberValue, sum}, nil
}

func (Builtin) Sub(vars ...Value) (Value, error) {
	if vars[0].typ != numberValue {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
	sum := vars[0].Number()
	for _, v := range vars[1:] {
		if v.typ == numberValue {
			sum -= v.Number()
		} else {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		}
	}
	return Value{numberValue, sum}, nil
}

func (Builtin) Mul(vars ...Value) (Value, error) {
	if vars[0].typ != numberValue {
		return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
	}
	sum := vars[0].Number()
	for _, v := range vars[1:] {
		if v.typ == numberValue {
			sum *= v.Number()
		} else {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		}
	}
	return Value{numberValue, sum}, nil
}

func (Builtin) Eq(vars ...Value) (Value, error) {
	for i := 1; i < len(vars); i++ {
		v1 := vars[i-1]
		v2 := vars[i]

		if v1.typ != v2.typ {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		} else if v1.typ == numberValue {
			if v1.Number() != v2.Number() {
				return False, nil
			}
		} else if v1.typ == stringValue {
			if v1.String() != v2.String() {
				return False, nil
			}
		} else {
			return Nil, fmt.Errorf("Unsupported argument type: %v", vars)
		}
	}
	return True, nil
}

func (Builtin) Gt(vars ...Value) (Value, error) {
	for i := 1; i < len(vars); i++ {
		v1 := vars[i-1]
		v2 := vars[i]
		if v1.typ != numberValue || v2.typ != numberValue {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		} else if !(v1.Number() > v2.Number()) {
			return False, nil
		}
	}
	return True, nil
}

func (Builtin) Lt(vars ...Value) (Value, error) {
	for i := 1; i < len(vars); i++ {
		v1 := vars[i-1]
		v2 := vars[i]
		if v1.typ != numberValue || v2.typ != numberValue {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		} else if !(v1.Number() < v2.Number()) {
			return False, nil
		}
	}
	return True, nil
}

func (Builtin) Gte(vars ...Value) (Value, error) {
	for i := 1; i < len(vars); i++ {
		v1 := vars[i-1]
		v2 := vars[i]
		if v1.typ != numberValue || v2.typ != numberValue {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		} else if !(v1.Number() >= v2.Number()) {
			return False, nil
		}
	}
	return True, nil
}

func (Builtin) Lte(vars ...Value) (Value, error) {
	for i := 1; i < len(vars); i++ {
		v1 := vars[i-1]
		v2 := vars[i]
		if v1.typ != numberValue || v2.typ != numberValue {
			return Nil, fmt.Errorf("Badly formatted arguments: %v", vars)
		} else if !(v1.Number() <= v2.Number()) {
			return False, nil
		}
	}
	return True, nil
}
