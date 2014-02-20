package tachyon

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Command interface {
	Run(pe *PlayEnv, args string) error
}

type Commands map[string]reflect.Type

var AvailableCommands Commands

var initAvailable sync.Once

func RegisterCommand(name string, cmd Command) {
	initAvailable.Do(func() {
		AvailableCommands = make(Commands)
	})

	ref := reflect.ValueOf(cmd)
	e := ref.Elem()

	AvailableCommands[name] = e.Type()
}

func (pe *PlayEnv) MakeCommand(task *Task, args string) (Command, error) {
	name := task.Command()

	t, ok := AvailableCommands[name]

	if !ok {
		return nil, fmt.Errorf("Unknown command: %s", name)
	}

	obj := reflect.New(t)

	sm, err := pe.ParseSimpleMap(args)

	if err == nil {
		for ik, iv := range task.Vars {
			exp, err := pe.ExpandVars(fmt.Sprintf("%v", iv))
			if err != nil {
				return nil, err
			}

			sm[ik] = exp
		}

		e := obj.Elem()

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			name := strings.ToLower(f.Name)
			required := false

			parts := strings.Split(f.Tag.Get("tachyon"), ",")

			switch len(parts) {
			case 0:
				// nothing
			case 1:
				name = parts[0]
			case 2:
				name = parts[0]
				switch parts[1] {
				case "required":
					required = true
				default:
					return nil, fmt.Errorf("Unsupported tag flag: %s", parts[1])
				}
			}

			if val, ok := sm[name]; ok {
				e.Field(i).Set(reflect.ValueOf(val))
			} else if required {
				return nil, fmt.Errorf("Missing value for %s", f.Name)
			}
		}
	}

	return obj.Interface().(Command), nil
}
