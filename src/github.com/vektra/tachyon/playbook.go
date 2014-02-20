package tachyon

import (
	"github.com/vektra/tachyon/lisp"
	"os"
	"path"
	"path/filepath"
)

type Vars map[string]interface{}

type VarsFiles []interface{}

type Notifications []string

type TaskData map[string]interface{}

type Play struct {
	Hosts      string
	Connection string

	Vars      Vars
	VarsFiles VarsFiles `yaml:"vars_files"`

	TaskDatas []TaskData `yaml:"tasks"`
	Tasks     Tasks      `yaml:"-"`

	HandlerDatas []TaskData `yaml:"handlers"`
	Handlers     Tasks      `yaml:"-"`

	baseDir string
}

type Playbook []*Play

func processTasks(datas []TaskData) Tasks {
	tasks := make(Tasks, len(datas))

	for idx, data := range datas {
		task := &Task{data: data}
		task.Init()

		tasks[idx] = task
	}

	return tasks
}

func LoadPlaybook(path string) (Playbook, error) {
	var p Playbook

	err := yamlFile(path, &p)

	if err != nil {
		return nil, err
	}

	baseDir, err := filepath.Abs(filepath.Dir(path))

	if err != nil {
		return nil, err
	}

	for _, play := range p {
		play.baseDir = baseDir
		play.Tasks = processTasks(play.TaskDatas)
		play.Handlers = processTasks(play.HandlerDatas)
	}

	return p, nil
}

func (p Playbook) Run(env *Environment) error {
	for _, play := range p {
		err := play.Run(env)

		if err != nil {
			return err
		}
	}

	return nil
}

func (play *Play) path(file string) string {
	return path.Join(play.baseDir, file)
}

func (play *Play) Run(env *Environment) error {
	env.report.StartTasks(play)

	pe := &PlayEnv{Vars: make(Vars), lispScope: lisp.NewScope()}
	pe.Init(env)

	pe.ImportVars(play.Vars)

	for _, file := range play.VarsFiles {
		switch file := file.(type) {
		case string:
			pe.ImportVarsFile(play.path(file))
			break
		case []interface{}:
			for _, ent := range file {
				exp, err := pe.ExpandVars(ent.(string))

				if err != nil {
					continue
				}

				epath := play.path(exp)

				if _, err := os.Stat(epath); err == nil {
					err = pe.ImportVarsFile(epath)

					if err != nil {
						return err
					}

					break
				}
			}
		}
	}

	for _, task := range play.Tasks {
		err := task.Run(env, pe)

		if err != nil {
			return err
		}
	}

	env.report.FinishTasks(play)

	pe.wait.Wait()

	env.report.StartHandlers(play)

	for _, task := range play.Handlers {
		if pe.ShouldRunHandler(task.Name()) {
			err := task.Run(env, pe)

			if err != nil {
				return err
			}
		}
	}

	env.report.FinishHandlers(play)

	return nil
}

func boolify(str string) bool {
	switch str {
	case "", "false", "no":
		return false
	default:
		return true
	}
}

func (task *Task) Run(env *Environment, pe *PlayEnv) error {
	if when := task.When(); when != "" {
		when, err := pe.ExpandVars(when)

		if err != nil {
			return err
		}

		if !boolify(when) {
			return nil
		}
	}

	str, err := pe.ExpandVars(task.Args())

	if err != nil {
		return err
	}

	cmd, err := pe.MakeCommand(task, str)

	if err != nil {
		return err
	}

	pe.report.StartTask(task, cmd, str)

	if task.Async() {
		asyncAction := &AsyncAction{Task: task}
		asyncAction.Init(pe)

		go func() {
			asyncAction.Finish(cmd.Run(pe, str))
		}()
	} else {
		err = cmd.Run(pe, str)

		pe.report.FinishTask(task, false)

		if err == nil {
			for _, x := range task.Notify() {
				pe.AddNotify(x)
			}
		}
	}

	return err
}
