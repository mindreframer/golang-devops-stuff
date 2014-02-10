package task

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestBoolFlag_DefType(t *testing.T) {
	f := NewBoolFlag("name", "usage")
	assert.Equal(t, `tasking.NewBoolFlag("name", "usage")`, f.DefType("tasking"))
}
