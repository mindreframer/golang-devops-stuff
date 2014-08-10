package tachyon

import (
	"fmt"
	"strings"
)

type Task struct {
	Play *Play
	File string

	data TaskData
	cmd  string
	args string
	Vars Vars

	IncludeVars Vars
	Paths       Paths
}

type TaskData map[string]interface{}

type Tasks []*Task

func AdhocTask(cmd, args string) *Task {
	return &Task{
		cmd:  cmd,
		args: args,
		data: TaskData{"name": "adhoc"},
		Vars: make(Vars),
	}
}

var cOptions = []string{
	"name", "action", "notify", "async", "poll",
	"when", "future", "register", "with_items",
}

func (t *Task) Init(env *Environment) error {
	t.Vars = make(Vars)

	for k, v := range t.data {
		found := false

		for _, i := range cOptions {
			if k == i {
				found = true
				break
			}
		}

		if !found {
			if t.cmd != "" {
				return fmt.Errorf("Duplicate command '%s', already: %s", k, t.cmd)
			}

			t.cmd = k
			if m, ok := v.(map[interface{}]interface{}); ok {
				for ik, iv := range m {
					t.Vars[fmt.Sprintf("%v", ik)] = Any(iv)
				}
			} else {
				t.args = fmt.Sprintf("%v", v)
			}
		}
	}

	if t.cmd == "" {
		act, ok := t.data["action"]
		if !ok {
			return fmt.Errorf("No action specified")
		}

		parts := strings.SplitN(fmt.Sprintf("%v", act), " ", 2)

		t.cmd = parts[0]

		if len(parts) == 2 {
			t.args = parts[1]
		}
	}

	t.Paths = env.Paths

	return nil
}

func (t *Task) Command() string {
	return t.cmd
}

func (t *Task) Args() string {
	return t.args
}

func (t *Task) Name() string {
	return t.data["name"].(string)
}

func (t *Task) Register() string {
	if v, ok := t.data["register"]; ok {
		return v.(string)
	}

	return ""
}

func (t *Task) Future() string {
	if v, ok := t.data["future"]; ok {
		return v.(string)
	}

	return ""
}

func (t *Task) When() string {
	if v, ok := t.data["when"]; ok {
		return v.(string)
	}

	return ""
}

func (t *Task) Notify() []string {
	var v interface{}
	var ok bool

	if v, ok = t.data["notify"]; !ok {
		return nil
	}

	var list []interface{}

	if list, ok = v.([]interface{}); !ok {
		return nil
	}

	out := make([]string, len(list))

	for i, x := range list {
		out[i] = x.(string)
	}

	return out
}

func (t *Task) Async() bool {
	_, ok := t.data["async"]

	return ok
}

func (t *Task) Items() []interface{} {
	if v, ok := t.data["with_items"]; ok {
		if a, ok := v.([]interface{}); ok {
			return a
		}
	}

	return nil
}
