package exiter

import (
	"os"
)

type Exiter interface {
	Exit(status int)
}

func New() Exiter {
	return &RealExiter{}
}

type RealExiter struct{}

func (e *RealExiter) Exit(status int) {
	os.Exit(status)
}
