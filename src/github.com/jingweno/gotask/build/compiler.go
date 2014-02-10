package build

import (
	"fmt"
	"github.com/jingweno/gotask/task"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type compiler struct {
	sourceDir string
	workDir   string
	TaskSet   *task.TaskSet
	isDebug   bool
}

func (c *compiler) Compile(outfile string) (execFile string, err error) {
	file, err := c.writeTaskMain(c.workDir, c.TaskSet)
	if err != nil {
		return
	}

	err = c.removeInstalledPkgs()
	if err != nil {
		return
	}

	execFile, err = c.compileTaskMain(c.sourceDir, file, outfile)
	return
}

func (c *compiler) removeInstalledPkgs() (err error) {
	pkgObj := c.TaskSet.PkgObj
	if pkgObj == "" {
		return
	}

	if c.isDebug {
		debugf("removing installed package %s", pkgObj)
	}
	err = os.RemoveAll(pkgObj)
	if err != nil {
		return
	}

	pkgDir := strings.TrimRight(pkgObj, ".a")
	if c.isDebug {
		debugf("removing installed package %s", pkgDir)
	}

	err = os.RemoveAll(pkgDir)

	return
}

func (c *compiler) writeTaskMain(work string, taskSet *task.TaskSet) (file string, err error) {
	// create task dir
	taskDir := filepath.Join(work, filepath.FromSlash(taskSet.ImportPath))
	err = os.MkdirAll(taskDir, 0777)
	if err != nil {
		return
	}

	// create main.go
	file = filepath.Join(taskDir, "main.go")
	f, err := os.Create(file)
	if err != nil {
		return
	}
	defer f.Close()

	if c.isDebug {
		debugf("writing task main to %s", file)
	}
	// write to main.go
	w := mainWriter{taskSet}
	err = w.Write(f)

	return
}

func (c *compiler) compileTaskMain(sourceDir, mainFile, outfile string) (exec string, err error) {
	taskDir := filepath.Dir(mainFile)

	err = os.Chdir(taskDir)
	if err != nil {
		return
	}

	// TODO: consider caching build
	compileCmd := []string{"go", "build", "--tags", "gotask"}
	if outfile != "" {
		if runtime.GOOS == "windows" {
			outfile = fmt.Sprintf("%s.exe", outfile)
		}
		exec = outfile
		compileCmd = append(compileCmd, "-o", outfile)
	}

	if c.isDebug {
		debugf("compiling tasks with `%s`", strings.Join(compileCmd, " "))
	}

	err = execCmd(compileCmd...)
	if err != nil {
		return
	}

	err = os.Chdir(sourceDir)
	if err != nil {
		return
	}

	// return if exec file has been assigned
	if exec != "" {
		return
	}

	// find exec file if it's not there
	files, err := ioutil.ReadDir(taskDir)
	if err != nil {
		return
	}

	execPrefix := filepath.Base(taskDir)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), execPrefix) {
			exec = filepath.Join(taskDir, file.Name())
			return
		}
	}

	err = fmt.Errorf("can't locate build executable for task main %s", mainFile)
	return
}
