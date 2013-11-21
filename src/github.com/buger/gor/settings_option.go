package gor

import (
	"fmt"
)

type MultiOption []string

func (h *MultiOption) String() string {
	return fmt.Sprint(*h)
}

func (h *MultiOption) Set(value string) error {
	*h = append(*h, value)
	return nil
}
