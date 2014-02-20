package lisp

func EvalString(line string, scope *Scope) (Value, error) {
	expanded, err := NewTokens(line).Expand()
	if err != nil {
		return Nil, err
	}
	parsed, err := expanded.Parse()
	if err != nil {
		return Nil, err
	}
	evaled, err := parsed.Eval(scope)
	if err != nil {
		return Nil, err
	}
	scope.Create("_", evaled)
	return evaled, nil
}
