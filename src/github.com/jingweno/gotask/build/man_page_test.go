package build

import (
	"testing"

	"github.com/bmizerany/assert"
	"github.com/jingweno/gotask/task"
)

func TestManPageParser_Parse(t *testing.T) {
	doc := `NAME
    say-hello - Say hello to current user

DESCRIPTION
    Print out hello to current user
    one more line

OPTIONS
    -n, --name=<NAME>
        Say hello to the given name
    -v, --verbose
        Run in verbose mode
    -g, --greeting=Hello
        Say hello with a custom type of greeting
`
	p := &manPageParser{doc}
	mp, err := p.Parse()

	assert.Equal(t, nil, err)
	assert.Equal(t, "say-hello", mp.Name)
	assert.Equal(t, "Say hello to current user", mp.Usage)
	assert.Equal(t, "Print out hello to current user\n   one more line", mp.Description)
	assert.Equal(t, 3, len(mp.Flags))

	stringFlag, ok := mp.Flags[0].(task.StringFlag)
	assert.Tf(t, ok, "Can't convert flag to task.StringFlag")
	assert.Equal(t, "n, name", stringFlag.Name)
	assert.Equal(t, "", stringFlag.Value)
	assert.Equal(t, "Say hello to the given name", stringFlag.Usage)

	boolFlag, ok := mp.Flags[1].(task.BoolFlag)
	assert.Tf(t, ok, "Can't convert flag to task.BoolFlag")
	assert.Equal(t, "v, verbose", boolFlag.Name)
	assert.Equal(t, "Run in verbose mode", boolFlag.Usage)

	stringFlag, ok = mp.Flags[2].(task.StringFlag)
	assert.Tf(t, ok, "Can't convert flag to task.StringFlag")
	assert.Equal(t, "g, greeting", stringFlag.Name)
	assert.Equal(t, "Hello", stringFlag.Value)
	assert.Equal(t, "Say hello with a custom type of greeting", stringFlag.Usage)

	doc = `Name
    say-hello - Say hello to current user

Description
    Print out hello to current user
`
	p = &manPageParser{doc}
	mp, err = p.Parse()

	assert.Equal(t, nil, err)
	assert.Equal(t, "", mp.Name)
	assert.Equal(t, "", mp.Usage)
	assert.Equal(t, "", mp.Description)
	assert.Equal(t, 0, len(mp.Flags))
}
