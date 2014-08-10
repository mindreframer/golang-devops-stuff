package lisp

import (
	"fmt"
	"strings"
)

type Value struct {
	typ valueType
	val interface{}
}

type Map interface {
	Get(key string) (Value, bool)
}

func (v Value) Interface() interface{} {
	return v.val
}

var Nil = Value{nilValue, nil}
var False = Value{symbolValue, "false"}
var True = Value{symbolValue, "true"}

type valueType uint8

const (
	nilValue valueType = iota
	symbolValue
	numberValue
	stringValue
	vectorValue
	procValue
	consValue
	mapValue
)

func NumberValue(n int64) Value {
	return Value{typ: numberValue, val: n}
}

func StringValue(s string) Value {
	return Value{typ: stringValue, val: s}
}

func MapValue(m Map) Value {
	return Value{typ: mapValue, val: m}
}

func (v Value) Eval(scope ScopedVars) (Value, error) {
	switch v.typ {
	case consValue:
		return v.Cons().Execute(scope)
	case symbolValue:
		sym := v.String()

		parts := strings.Split(sym, ".")

		var (
			v  Value
			ok bool
		)

		if len(parts) == 1 {
			v, ok = scope.Get(sym)
		} else {
			v, ok = scope.Get(parts[0])

			for _, sub := range parts[1:] {
				if v.typ != mapValue {
					return Nil, fmt.Errorf("Variable '%s' is not a map (%v)", parts[0], v)
				}

				v, ok = v.Interface().(Map).Get(sub)
			}
		}

		if ok {
			return v, nil
		} else if sym == "true" || sym == "false" {
			return Value{symbolValue, sym}, nil
		} else {
			return Nil, fmt.Errorf("Unbound variable: %v", sym)
		}
	default:
		return v, nil
	}
}

func (v Value) String() string {
	switch v.typ {
	case numberValue:
		return fmt.Sprintf("%d", v.val.(int64))
	case nilValue:
		return "()"
	default:
		return fmt.Sprintf("%v", v.val)
	}
}

func (v Value) Inspect() string {
	switch v.typ {
	case stringValue:
		return fmt.Sprintf(`"%v"`, v.val)
	case vectorValue:
		return v.val.(Vector).Inspect()
	default:
		return v.String()
	}
}

func (v Value) Cons() Cons {
	if v.typ == consValue {
		return *v.val.(*Cons)
	} else {
		return Cons{&v, &Nil}
	}
}

func (v Value) Number() int64 {
	return v.val.(int64)
}
