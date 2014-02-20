package tachyon

import (
	"fmt"
	"strings"
)

type strmap map[string]interface{}

type Task struct {
	data TaskData
	cmd  string
	args string
	Vars strmap
}

type Tasks []*Task

var cOptions = []string{"name", "action", "notify", "async", "poll",
	"when"}

func (t *Task) Init() error {
	t.Vars = make(strmap)

	for k, v := range t.data {
		found := false

		for _, i := range cOptions {
			if k == i {
				found = true
				break
			}
		}

		if !found {
			t.cmd = k
			if m, ok := v.(map[interface{}]interface{}); ok {
				for ik, iv := range m {
					t.Vars[fmt.Sprintf("%v", ik)] = iv
				}
			}

			t.args = fmt.Sprintf("%v", v)
		}
	}

	if t.cmd == "" {
		act, ok := t.data["action"]
		if !ok {
			return fmt.Errorf("No action specified")
		}

		parts := strings.SplitN(fmt.Sprintf("%v", act), " ", 2)

		t.cmd = parts[0]
		t.args = parts[1]
	}

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
