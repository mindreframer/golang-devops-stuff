package tachyon

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/vektra/tachyon/lisp"
	"strings"
	"unicode"
)

var cTemplateStart = []byte(`{{`)
var cTemplateEnd = []byte(`}}`)
var cExprStart = []byte(`$(`)
var cExprEnd = []byte(`)`)

var eUnclosedTemplate = errors.New("Unclosed template")
var eUnclosedExpr = errors.New("Unclosed lisp expression")

func expandTemplates(s Scope, args string) (string, error) {
	a := []byte(args)

	var buf bytes.Buffer

	for {
		idx := bytes.Index(a, cTemplateStart)

		if idx == -1 {
			buf.Write(a)
			break
		}

		buf.Write(a[:idx])

		in := a[idx+2:]

		fin := bytes.Index(in, cTemplateEnd)

		if fin == -1 {
			return "", eUnclosedTemplate
		}

		name := bytes.TrimSpace(in[:fin])

		parts := strings.Split(string(name), ".")

		var (
			val Value
			ok  bool
		)

		if len(parts) == 1 {
			val, ok = s.Get(string(name))
		} else {
			cur := parts[0]

			val, ok = s.Get(cur)

			for _, sub := range parts[1:] {
				m, ok := val.(Map)
				if !ok {
					m, ok = val.Read().(Map)
					if !ok {
						return "", fmt.Errorf("Variable '%s' is not a Map (%T)", cur, val.Read())
					}
				}

				val, ok = m.Get(sub)
				if !ok {
					return "", fmt.Errorf("Variable '%s' has no key '%s'", cur, sub)
				}
				cur = sub
			}
		}

		if ok {
			switch val := val.Read().(type) {
			case int64, int:
				buf.WriteString(fmt.Sprintf("%d", val))
			default:
				buf.WriteString(fmt.Sprintf("%s", val))
			}

			a = in[fin+2:]
		} else {
			return "", fmt.Errorf("Undefined variable: %s", string(name))
		}
	}

	return buf.String(), nil
}

func findExprClose(buf []byte) int {
	opens := 0

	for idx, r := range buf {
		switch r {
		case ')':
			opens--

			if opens == 0 {
				return idx
			}

		case '(':
			opens++
		}
	}

	return -1
}

func varChar(r rune) bool {
	if unicode.IsLetter(r) {
		return true
	}
	if unicode.IsDigit(r) {
		return true
	}
	if r == '_' {
		return true
	}
	return false
}

func inferValue(val Value) lisp.Value {
	switch lv := val.Read().(type) {
	case int:
		return lisp.NumberValue(int64(lv))
	case int32:
		return lisp.NumberValue(int64(lv))
	case int64:
		return lisp.NumberValue(lv)
	case string:
		return lisp.StringValue(lv)
	case *Result:
		return lisp.MapValue(&lispResult{lv})
	default:
	}

	return lisp.StringValue(fmt.Sprintf("%s", val.Read()))
}

type lispResult struct {
	res *Result
}

func (lr *lispResult) Get(key string) (lisp.Value, bool) {
	v, ok := lr.res.Get(key)

	if !ok {
		return lisp.Nil, false
	}

	return inferValue(v), true
}

type lispInferredScope struct {
	Scope Scope
}

func (s lispInferredScope) Get(key string) (lisp.Value, bool) {
	val, ok := s.Scope.Get(key)

	if !ok {
		return lisp.Nil, false
	}

	return inferValue(val), true
}

func (s lispInferredScope) Set(key string, v lisp.Value) lisp.Value {
	s.Scope.Set(key, v.Interface())
	return v
}

func (s lispInferredScope) Create(key string, v lisp.Value) lisp.Value {
	s.Scope.Set(key, v.Interface())
	return v
}

var cDollar = []byte(`$`)

func ExpandVars(s Scope, args string) (string, error) {
	args, err := expandTemplates(s, args)

	if err != nil {
		return "", err
	}

	a := []byte(args)

	var buf bytes.Buffer

	for {
		idx := bytes.Index(a, cDollar)

		if idx == -1 {
			buf.Write(a)
			break
		} else if a[idx+1] == '(' {
			buf.Write(a[:idx])

			in := a[idx+1:]

			fin := findExprClose(in)

			if fin == -1 {
				return "", eUnclosedExpr
			}

			sexp := in[:fin+1]

			ls := lispInferredScope{s}

			val, err := lisp.EvalString(string(sexp), ls)

			if err != nil {
				return "", err
			}

			buf.WriteString(val.String())
			a = in[fin+1:]
		} else {
			buf.Write(a[:idx])

			in := a[idx+1:]

			fin := 0

			for fin < len(in) {
				if !varChar(rune(in[fin])) {
					break
				}
				fin++
			}

			if val, ok := s.Get(string(in[:fin])); ok {
				switch val := val.Read().(type) {
				case int64, int:
					buf.WriteString(fmt.Sprintf("%d", val))
				default:
					buf.WriteString(fmt.Sprintf("%s", val))
				}

				a = in[fin:]
			} else {
				return "", fmt.Errorf("Undefined variable: %s", string(in[:fin]))
			}
		}
	}

	return buf.String(), nil
}
