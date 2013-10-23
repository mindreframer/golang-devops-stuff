package errplane

import (
	"strings"
)

func validCharacter(ch rune) bool {
	return ch >= 'a' && ch <= 'z' ||
		ch >= 'A' && ch <= 'Z' ||
		ch >= '0' && ch <= '9' ||
		ch == '-' || ch == '_' ||
		ch == '.'
}

func notValidCharacter(ch rune) bool {
	return !validCharacter(ch)
}

func isValidMetricName(name string) bool {
	return len(name) <= 255 &&
		strings.IndexFunc(name, notValidCharacter) == -1
}
