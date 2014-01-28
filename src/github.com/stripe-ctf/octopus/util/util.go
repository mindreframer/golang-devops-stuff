package util

import (
	"fmt"
	"github.com/stripe-ctf/octopus/log"
	"os"
	"strings"
)

func EnsureAbsent(path string) {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
}

func FmtOutput(out []byte) string {
	o := string(out)
	if strings.ContainsAny(o, "\n") {
		return fmt.Sprintf(`"""
%s"""`, o)
	} else {
		return fmt.Sprintf("%#v", o)
	}
}
