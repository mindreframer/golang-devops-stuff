package command

import (
	"fmt"
)

type RetryError struct {
	Seconds int
}

func (r *RetryError) Error() string {
	return fmt.Sprintf("Retry(seconds=%d)", r.Seconds)
}

type AllUpstreamsDownError struct {
}

func (r *AllUpstreamsDownError) Error() string {
	return fmt.Sprintf("AllUpstreamsDown")
}
