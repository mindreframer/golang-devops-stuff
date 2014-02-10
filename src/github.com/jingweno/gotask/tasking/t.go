package tasking

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/kballard/go-shellquote"
	"os"
	"strings"
	"sync"
)

// Flags can be used to retrieve parsed command-line options.
type Flags struct {
	C *cli.Context
}

// Bool looks up the value of a bool flag, returns false if no bool flag exists
func (f Flags) Bool(name string) bool {
	return f.C.Bool(name)
}

// String looks up the value of a string flag, returns "" if no string flag exists
func (f Flags) String(name string) string {
	return f.C.String(name)
}

// T is a type that is passed through to each task function.
// T can be used to retrieve context-specific Args and parsed command-line Flags.
type T struct {
	mu     sync.RWMutex
	Args   []string // command-line arguments
	Flags  Flags    // command-line options
	output []string
	failed bool
}

// Exec runs the system command. If multiple arguments are given, they're concatenated to one command.
//
// Example:
//   t.Exec("ls -ltr")
//   t.Exec("ls", FILE1, FILE2)
func (t *T) Exec(cmd ...string) (err error) {
	toRun := strings.Join(cmd, " ")
	input, err := shellquote.Split(toRun)
	if err != nil {
		return
	}
	err = execCmd(input)

	return
}

// Fail marks the task as having failed but continues execution.
func (t *T) Fail() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed = true
}

// Failed checks if the task has failed
func (t *T) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

// Fatal is equivalent to Error followed by a call to os.Exit(1).
func (t *T) Fatal(args ...interface{}) {
	t.Error(args...)
	os.Exit(1)
}

// Fatalf is equivalent to Errorf followed by a call to os.Exit(1).
func (t *T) Fatalf(format string, args ...interface{}) {
	t.Errorf(format, args...)
	os.Exit(1)
}

// Log formats its arguments using default formatting, analogous to Println.
func (t *T) Log(args ...interface{}) {
	fmt.Println(args...)
}

// Logf formats its arguments according to the format, analogous to Printf.
func (t *T) Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Error is equivalent to Log followed by Fail.
func (t *T) Error(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	t.Fail()
}

// Errorf is equivalent to Logf followed by Fail.
func (t *T) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	t.Fail()
}
