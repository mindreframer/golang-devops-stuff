package lisp

import "testing"
import "fmt"

func TestEval(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"()", "()"},
		{"42", "42"},
		{"1 2 3", "3"},
		{"(+ 42 13)", "55"},
		{"(+ (+ 1 2 3) 4)", "10"},
		{"(quote (1 2 3))", "(1 2 3)"},
		{"(quote (1 (+ 1 2) 3))", "(1 (+ 1 2) 3)"},
		{"(quote hej)", "hej"},
		{"(cons 1 2)", "(1 . 2)"},
		{"(car (cons 1 2))", "1"},
		{"(cdr (cons 1 2))", "2"},
		{"(cons 1 ())", "(1)"},
		{"(cons 1 :(2))", "(1 2)"},
		{":hej", "hej"},
		{"::hej", "(quote hej)"},
		{":(hej hopp)", "(hej hopp)"},
		{"(quote (hej))", "(hej)"},
		{"(if true (+ 1 1) 3)", "2"},
		{"(if false 42 1)", "1"},
		{"(if false 42)", "()"},
		{"(begin (define x) (if x 1 2))", "2"},
		{"(define r 3)", "r"},
		{"(begin 5 (+ 3 4))", "7"},
		{"(begin (define p 3) (+ 39 p))", "42"},
		{"(begin (define p 3) (set! p 4) (+ 1 p))", "5"},
		{"(begin (define p 3) (set! p (+ 1 1)) p)", "2"},
		{"(begin (define pi (+ 3 14)) pi)", "17"},
		{"((lambda (a) (+ a 1)) 42)", "43"},
		{"(begin (define p 10) p)", "10"},
		{"(begin (define inc (lambda (a) (+ a 1))) (inc 42))", "43"},
		// {"(define a 10) ((lambda () (define a 20))) a", "10"},
		{"(define a 0) ((lambda () (set! a 10))) a", "10"},
		{"((lambda (i) i) (+ 5 5))", "10"},
		{"(define inc ((lambda () (begin (define a 0) (lambda () (set! a (+ a 1))))))) (inc) (inc)", "2"},
		{"(define fact (lambda (n) (if (<= n 1) 1 (* n (fact (- n 1)))))) (fact 20)", "2432902008176640000"},
	}

	for _, test := range tests {
		if actual, err := EvalString(test.in, scope); err != nil {
			t.Error(err)
		} else if fmt.Sprintf("%v", actual) != test.out {
			t.Errorf("Eval \"%v\" gives \"%v\", want \"%v\"", test.in, actual, test.out)
		}
	}
}

func TestEvalFailures(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"hello", "Unbound variable: hello"},
		{"(set! undefined 42)", "Unbound variable: undefined"},
		{"(lambda (a))", "Ill-formed special form: (lambda (a))"},
		{"(1 2 3)", "The object 1 is not applicable"},
		{"(1", "List was opened but not closed"},
		{"(set! a)", "Ill-formed special form: (set! a)"},
	}

	for _, test := range tests {
		if _, err := EvalString(test.in, scope); err == nil || err.Error() != test.out {
			t.Errorf("Parse('%v'), want error '%v', got '%v'", test.in, test.out, err)
		}
	}
}
