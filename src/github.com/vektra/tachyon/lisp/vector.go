package lisp

import (
	"fmt"
	"strings"
)

type Vector []Value

func (s Vector) String() string {
	var arr []string
	for _, v := range s {
		arr = append(arr, v.String())
	}
	return fmt.Sprintf(`[%v]`, strings.Join(arr, " "))
}

func (s Vector) Inspect() string {
	var arr []string
	for _, v := range s {
		arr = append(arr, v.Inspect())
	}
	return fmt.Sprintf(`[%v]`, strings.Join(arr, " "))
}
