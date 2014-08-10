package lisp

import (
	"fmt"
	"regexp"
	"strconv"
)

type Tokens []*Token

type tokenType uint8

type Token struct {
	typ tokenType
	val string
}

type Pattern struct {
	typ    tokenType
	regexp *regexp.Regexp
}

// func (t Token) String() string {
// return fmt.Sprintf("%v", t.val)
// }

func (t *Token) String() string {
	return fmt.Sprintf("%v", t.val)
}

const (
	whitespaceToken tokenType = iota
	commentToken
	stringToken
	numberToken
	openToken
	closeToken
	symbolToken
)

func (t *Token) Type() string {
	switch t.typ {
	case commentToken:
		return "comment"
	case stringToken:
		return "string"
	case numberToken:
		return "number"
	case openToken:
		return "open"
	case closeToken:
		return "close"
	case symbolToken:
		return "symbol"
	default:
		return "unknown"
	}
}

func patterns() []Pattern {
	return []Pattern{
		{whitespaceToken, regexp.MustCompile(`^\s+`)},
		{commentToken, regexp.MustCompile(`^;.*`)},
		{stringToken, regexp.MustCompile(`^("(\\.|[^"])*")`)},
		{numberToken, regexp.MustCompile(`^((([0-9]+)?\.)?[0-9]+)`)},
		{openToken, regexp.MustCompile(`^(\()`)},
		{closeToken, regexp.MustCompile(`^(\))`)},
		{symbolToken, regexp.MustCompile(`^(:|[^\s();]+)`)},
	}
}

func NewTokens(program string) (tokens Tokens) {
	for pos := 0; pos < len(program); {
		for _, pattern := range patterns() {
			if matches := pattern.regexp.FindStringSubmatch(program[pos:]); matches != nil {
				if len(matches) > 1 {
					tokens = append(tokens, &Token{pattern.typ, matches[1]})
				}
				pos = pos + len(matches[0])
				break
			}
		}
	}
	return
}

// Expand until there are no more expansions to do
func (tokens Tokens) Expand() (result Tokens, err error) {
	var updated bool
	for i := 0; i < len(tokens); i++ {
		var start int
		quote := Token{symbolToken, ":"}
		if *tokens[i] != quote {
			result = append(result, tokens[i])
		} else {
			updated = true
			for start = i + 1; *tokens[start] == quote; start++ {
				result = append(result, tokens[start])
			}
			if tokens[i+1].typ == openToken {
				if i, err = tokens.findClose(start + 1); err != nil {
					return nil, err
				}
			} else {
				i = start
			}
			result = append(result, &Token{openToken, "("}, &Token{symbolToken, "quote"})
			result = append(result, tokens[start:i+1]...)
			result = append(result, &Token{closeToken, ")"})
		}
	}
	if updated {
		result, err = result.Expand()
	}
	return
}

func (tokens Tokens) Parse() (cons Cons, err error) {
	var pos int
	var current *Cons
	for pos < len(tokens) {
		if current == nil {
			cons = Cons{&Nil, &Nil}
			current = &cons
		} else {
			previous_current := current
			current = &Cons{&Nil, &Nil}
			previous_current.cdr = &Value{consValue, current}
		}
		t := tokens[pos]
		switch t.typ {
		case numberToken:
			if i, err := strconv.ParseInt(t.val, 10, 0); err != nil {
				err = fmt.Errorf("Failed to convert number: %v", t.val)
			} else {
				current.car = &Value{numberValue, i}
				pos++
			}
		case stringToken:
			current.car = &Value{stringValue, t.val[1 : len(t.val)-1]}
			pos++
		case symbolToken:
			current.car = &Value{symbolValue, t.val}
			pos++
		case openToken:
			var nested Cons
			start := pos + 1
			var end int
			if end, err = tokens.findClose(start); err != nil {
				return
			}
			if start == end {
				current.car = &Nil
			} else {
				if nested, err = tokens[start:end].Parse(); err != nil {
					return
				}
				current.car = &Value{consValue, &nested}
			}
			pos = end + 1
		case closeToken:
			err = fmt.Errorf("List was closed but not opened")
		}
	}
	return
}

func (t Tokens) findClose(start int) (int, error) {
	depth := 1
	for i := start; i < len(t); i++ {
		t := t[i]
		switch t.typ {
		case openToken:
			depth++
		case closeToken:
			depth--
		}
		if depth == 0 {
			return i, nil
		}
	}
	return 0, fmt.Errorf("List was opened but not closed")
}
