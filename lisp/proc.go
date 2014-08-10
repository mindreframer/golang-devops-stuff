package lisp

import "fmt"

type Proc struct {
	params Vector
	body   Cons
	scope  ScopedVars
}

func (p Proc) String() string {
	return "<Procedure>"
}

func (p Proc) Call(scope ScopedVars, params Vector) (val Value, err error) {
	if len(p.params) == len(params) {
		scope = p.scope
		for i, name := range p.params {
			scope.Create(name.String(), params[i])
		}
		val, err = p.body.Eval(scope)
	} else {
		err = fmt.Errorf("%v has been called with %v arguments; it requires exactly %v arguments", p, len(params), len(p.params))
	}
	return
}
