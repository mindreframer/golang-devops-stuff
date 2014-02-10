package build

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func isFileExist(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func execCmd(cmd ...string) error {
	binary, lookErr := exec.LookPath(cmd[0])
	if lookErr != nil {
		return fmt.Errorf("command not found: %s", cmd[0])
	}

	c := exec.Command(binary, cmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

func dasherize(s string) string {
	r := regexp.MustCompile("([A-Z\\d]+)([A-Z][a-z])")
	s = r.ReplaceAllString(s, "$1-$2")

	r = regexp.MustCompile("([a-z\\d])([A-Z])")
	s = r.ReplaceAllString(s, "$1-$2")

	s = strings.Replace(s, "_", "-", -1)
	s = strings.ToLower(s)

	return s
}

func debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("debug - %s", msg)
}
