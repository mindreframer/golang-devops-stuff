package gor

import (
	"fmt"
	"strings"
)

type HTTPMethods []string

func (h *HTTPMethods) String() string {
	return fmt.Sprint(*h)
}

func (h *HTTPMethods) Set(value string) error {
	*h = append(*h, strings.ToUpper(value))
	return nil
}

func (h *HTTPMethods) Contains(value string) bool {
	for _, method := range *h {
		if value == method {
			return true
		}
	}
	return false
}
