package build

import (
	"fmt"
	"github.com/jingweno/gotask/task"
	"go/ast"
	"go/build"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

func NewParser() *parser {
	return &parser{}
}

type parser struct{}

func (l *parser) Parse(dir string) (taskSet *task.TaskSet, err error) {
	dir, err = expandPath(dir)
	if err != nil {
		return
	}

	importPath, err := findImportPath(dir)
	if err != nil {
		return
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "gotask")
	p, e := ctx.Import(importPath, dir, 0)
	if e != nil {
		// allow no task files found
		if _, ok := e.(*build.NoGoError); !ok {
			err = e
			return
		}
	}

	// gather task files including those are ignored
	tasks, err := loadTasks(dir, p.GoFiles)
	if err != nil {
		return
	}

	name := p.Name
	if name == "" {
		name = filepath.Base(p.Dir)
	}

	// fix import path on Windows
	importPath = strings.Replace(p.ImportPath, "\\", "/", -1)

	taskSet = &task.TaskSet{
		Name:       name,
		Dir:        p.Dir,
		PkgObj:     p.PkgObj,
		ImportPath: importPath,
		Tasks:      tasks,
	}

	return
}

func expandPath(path string) (expanded string, err error) {
	expanded, err = filepath.Abs(path)
	if err != nil {
		return
	}

	if !isFileExist(expanded) {
		err = fmt.Errorf("Path %s does not exist", expanded)
		return
	}

	return
}

func findImportPath(dir string) (importPath string, err error) {
	p, e := build.ImportDir(dir, 0)
	if e != nil {
		// tasks maybe ignored for build
		if _, ok := e.(*build.NoGoError); !ok || p.ImportPath == "" {
			err = e
			return
		}
	}
	if err != nil {
		return
	}

	importPath = p.ImportPath
	return
}

func loadTasks(dir string, files []string) (tasks []task.Task, err error) {
	taskFiles := filterTaskFiles(files)
	for _, taskFile := range taskFiles {
		ts, e := parseTasks(filepath.Join(dir, taskFile))
		if e != nil {
			err = e
			return
		}

		tasks = append(tasks, ts...)
	}

	return
}

func filterTaskFiles(files []string) (taskFiles []string) {
	for _, f := range files {
		if isTaskFile(f, "_task.go") {
			taskFiles = append(taskFiles, f)
		}
	}

	return
}

func parseTasks(filename string) (tasks []task.Task, err error) {
	taskFileSet := token.NewFileSet()
	f, err := goparser.ParseFile(taskFileSet, filename, nil, goparser.ParseComments)
	if err != nil {
		return
	}

	for _, d := range f.Decls {
		n, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if n.Recv != nil {
			continue
		}

		actionName := n.Name.String()
		if isTask(actionName, "Task") {
			p := &manPageParser{n.Doc.Text()}
			mp, e := p.Parse()
			if e != nil {
				continue
			}

			if mp.Name == "" {
				mp.Name = convertActionNameToTaskName(actionName)
			}

			t := task.Task{Name: mp.Name, ActionName: actionName, Usage: mp.Usage, Description: mp.Description, Flags: mp.Flags}
			tasks = append(tasks, t)
		}
	}

	return
}

func isTaskFile(name, suffix string) bool {
	if strings.HasSuffix(name, suffix) {
		return true
	}

	return false
}

func isTask(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Task" is ok
		return true
	}

	rune, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(rune)
}

func convertActionNameToTaskName(s string) string {
	n := strings.TrimPrefix(s, "Task")
	return dasherize(n)
}
