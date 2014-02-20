package lisp

import (
	"fmt"
	"reflect"
	"strings"
)

type Cons struct {
	car *Value
	cdr *Value
}

func (c Cons) Eval(scope *Scope) (val Value, err error) {
	if c.List() {
		if v, err := c.car.Eval(scope); err != nil {
			return Nil, err
		} else if *c.cdr == Nil {
			return v, nil
		} else {
			return c.cdr.Cons().Eval(scope)
		}
	} else {
		return Value{consValue, c}, nil
	}
}

func (cons Cons) Execute(scope *Scope) (Value, error) {
	if !cons.List() {
		return Nil, fmt.Errorf("Combination must be a proper list: %v", cons)
	}
	switch cons.car.String() {
	case "quote":
		return cons.quoteForm(scope)
	case "if":
		return cons.ifForm(scope)
	case "set!":
		return cons.setForm(scope)
	case "define":
		return cons.defineForm(scope)
	case "lambda":
		return cons.lambdaForm(scope)
	case "begin":
		return cons.beginForm(scope)
	default:
		if cons.isBuiltin() {
			return cons.runBuiltin(scope)
		} else {
			return cons.procForm(scope)
		}
	}
}

func (c Cons) List() bool {
	return c.cdr.typ == consValue || c.cdr.typ == nilValue
}

func (c Cons) Map(f func(v Value) (Value, error)) ([]Value, error) {
	result := make([]Value, 0)
	if value, err := f(*c.car); err != nil {
		return nil, err
	} else {
		result = append(result, value)
	}
	if *c.cdr != Nil {
		if values, err := c.cdr.Cons().Map(f); err != nil {
			return nil, err
		} else {
			result = append(result, values...)
		}
	}
	return result, nil
}

func (c Cons) Len() int {
	l := 0
	if *c.car != Nil {
		l++
		if *c.cdr != Nil {
			l += c.cdr.Cons().Len()
		}
	}
	return l
}

func (c Cons) Vector() Vector {
	v, _ := c.Map(func(v Value) (Value, error) {
		return v, nil
	})
	return v
}

func (c Cons) String() string {
	s := strings.Join(c.Stringify(), " ")
	return fmt.Sprintf(`(%v)`, s)
}

func (c Cons) Stringify() []string {
	result := make([]string, 0)
	result = append(result, c.car.String())
	switch c.cdr.typ {
	case nilValue:
	case consValue:
		result = append(result, c.cdr.Cons().Stringify()...)
	default:
		result = append(result, ".", c.cdr.String())
	}
	return result
}

func (cons Cons) procForm(scope *Scope) (val Value, err error) {
	if val, err = cons.car.Eval(scope); err == nil {
		if val.typ == procValue {
			var args Vector
			if args, err = cons.cdr.Cons().Map(func(v Value) (Value, error) {
				return v.Eval(scope)
			}); err != nil {
				return
			} else {
				val, err = val.val.(Proc).Call(scope, args)
			}
		} else {
			err = fmt.Errorf("The object %v is not applicable", val)
		}
	}
	return
}

func (cons Cons) beginForm(scope *Scope) (val Value, err error) {
	return cons.cdr.Cons().Eval(scope)
}

func (cons Cons) setForm(scope *Scope) (val Value, err error) {
	expr := cons.Vector()
	if len(expr) == 3 {
		key := expr[1].String()
		if _, ok := scope.Get(key); ok {
			val, err = expr[2].Eval(scope)
			if err == nil {
				scope.Set(key, val)
			}
		} else {
			err = fmt.Errorf("Unbound variable: %v", key)
		}
	} else {
		err = fmt.Errorf("Ill-formed special form: %v", cons)
	}
	return
}

func (cons Cons) ifForm(scope *Scope) (val Value, err error) {
	expr := cons.Vector()
	val = Nil
	if len(expr) < 3 || len(expr) > 4 {
		err = fmt.Errorf("Ill-formed special form: %v", expr)
	} else {
		r, err := expr[1].Eval(scope)
		if err == nil {
			if !(r.typ == symbolValue && r.String() == "false") && r != Nil && len(expr) > 2 {
				val, err = expr[2].Eval(scope)
			} else if len(expr) == 4 {
				val, err = expr[3].Eval(scope)
			}
		}
	}
	return
}

func (cons Cons) lambdaForm(scope *Scope) (val Value, err error) {
	if cons.cdr.typ == consValue {
		lambda := cons.cdr.Cons()
		if (lambda.car.typ == consValue || lambda.car.typ == nilValue) && lambda.cdr.typ == consValue {
			params := lambda.car.Cons().Vector()
			val = Value{procValue, Proc{params, lambda.cdr.Cons(), scope.Dup()}}
		} else {
			err = fmt.Errorf("Ill-formed special form: %v", cons)
		}
	} else {
		err = fmt.Errorf("Ill-formed special form: %v", cons)
	}
	return
}

func (cons Cons) quoteForm(scope *Scope) (val Value, err error) {
	if cons.cdr != nil {
		if *cons.cdr.Cons().cdr == Nil {
			val = *cons.cdr.Cons().car
		} else {
			val = Value{consValue, cons}
		}
	} else {
		err = fmt.Errorf("Ill-formed special form: %v", cons)
	}
	return
}

func (cons Cons) defineForm(scope *Scope) (val Value, err error) {
	expr := cons.Vector()
	if len(expr) >= 2 && len(expr) <= 3 {
		if expr[1].typ == symbolValue {
			key := expr[1].String()
			if len(expr) == 3 {
				var i Value
				if i, err = expr[2].Eval(scope); err == nil {
					scope.Create(key, i)
				}
			} else {
				scope.Create(key, Nil)
			}
			return expr[1], err
		}
	}
	return Nil, fmt.Errorf("Ill-formed special form: %v", expr)
}

func (cons Cons) isBuiltin() bool {
	s := cons.car.String()
	if _, ok := builtin_commands[s]; ok {
		return true
	}
	return false
}

func (cons Cons) runBuiltin(scope *Scope) (val Value, err error) {
	cmd := builtin_commands[cons.car.String()]
	vars, err := cons.cdr.Cons().Map(func(v Value) (Value, error) {
		return v.Eval(scope)
	})

  if err != nil {
    return Nil, err
  }

	values := []reflect.Value{}
	for _, v := range vars {
		values = append(values, reflect.ValueOf(v))
	}
	result := reflect.ValueOf(&builtin).MethodByName(cmd).Call(values)
	val = result[0].Interface().(Value)
	err, _ = result[1].Interface().(error)
	return
}
