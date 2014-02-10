package build

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestDasherize(t *testing.T) {
	s := dasherize("HelloWorld")
	assert.Equal(t, "hello-world", s)

	s = dasherize("Helloworld")
	assert.Equal(t, "helloworld", s)

	s = dasherize("hello_world")
	assert.Equal(t, "hello-world", s)
}
