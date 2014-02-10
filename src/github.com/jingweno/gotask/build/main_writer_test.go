package build

import (
	"bytes"
	"github.com/bmizerany/assert"
	"github.com/jingweno/gotask/task"
	"strings"
	"testing"
)

func TestMainWriter_Write(t *testing.T) {
	var out bytes.Buffer
	b := mainWriter{
		&task.TaskSet{
			ImportPath: "github.com/jingweno/gotask/examples",
			Tasks: []task.Task{
				{
					Name:        "HelloWorld",
					ActionName:  "TaskHelloWorld",
					Usage:       "Say Hello world",
					Description: "Print out Hello World",
					Flags: []task.Flag{
						task.NewBoolFlag("v, verbose", "Run in verbose mode"),
					},
				},
			},
		},
	}
	b.Write(&out)

	assert.Tf(t, strings.Contains(out.String(), `_task "github.com/jingweno/gotask/examples"`), "%v", out.String())
	assert.Tf(t, strings.Contains(out.String(), `Name: "HelloWorld"`), "%v", out.String())
	assert.Tf(t, strings.Contains(out.String(), `Usage: "Say Hello world"`), "%v", out.String())
	assert.Tf(t, strings.Contains(out.String(), `Description: "Print out Hello World`), "%v", out.String())
	assert.Tf(t, strings.Contains(out.String(), `task.NewBoolFlag("v, verbose", "Run in verbose mode")`), "%v", out.String())
}
