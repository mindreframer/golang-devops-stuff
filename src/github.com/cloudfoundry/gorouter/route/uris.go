package route

import (
	"strings"
)

type Uri string

func (u Uri) ToLower() Uri {
	return Uri(strings.ToLower(string(u)))
}
