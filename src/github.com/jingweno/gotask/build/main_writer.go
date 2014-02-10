package build

import (
	"github.com/jingweno/gotask/task"
	"io"
	"text/template"
)

type mainWriter struct {
	TaskSet *task.TaskSet
}

func (w *mainWriter) Write(wr io.Writer) (err error) {
	err = taskmainTmpl.Execute(wr, w.TaskSet)
	return
}

var taskmainTmpl = template.Must(template.New("main").Parse(`
package main

import (
  "os"
  "github.com/jingweno/gotask/task"
{{if .HasTasks}}
  _task "{{.ImportPath}}"
{{end}}
)

var tasks = []task.Task{
{{range .Tasks}}
  {
    Name: {{.Name | printf "%q" }},
    Usage: {{.Usage | printf "%q"}},
    Description: {{.Description | printf "%q"}},
    Action: _task.{{.ActionName}},
    Flags: []task.Flag{
      {{range .Flags}}
        {{.DefType "task"}},
      {{end}}
    },
  },
{{end}}
}

var taskSet = task.TaskSet{
  Name: {{.Name | printf "%q" }},
  Dir: {{.Dir | printf "%q" }},
  PkgObj: {{.PkgObj | printf "%q" }},
  ImportPath: {{.ImportPath | printf "%q" }},
  Tasks: tasks,
}

func main() {
  task.Run(&taskSet, os.Args)
}
`))
