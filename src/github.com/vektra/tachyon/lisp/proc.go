package lisp

import "fmt"

type Proc struct {
	params Vector
	body   Cons
	scope  *Scope
}

func (p Proc) String() string {
	return "<Procedure>"
}

func (p Proc) Call(scope *Scope, params Vector) (val Value, err error) {
	if len(p.params) == len(params) {
		scope = p.scope
		scope.AddEnv()
		for i, name := range p.params {
			scope.Create(name.String(), params[i])
		}
		val, err = p.body.Eval(scope)
		scope.DropEnv()
	} else {
		err = fmt.Errorf("%v has been called with %v arguments; it requires exactly %v arguments", p, len(params), len(p.params))
	}
	return
}
