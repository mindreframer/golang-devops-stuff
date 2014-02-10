package cli

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
	"path/filepath"
)

var generateFlag = cli.BoolFlag{Name: "generate, g", Usage: "generate a task scaffolding named pkg_task.go"}

func generateNewTask() (fileName string, err error) {
	sourceDir, err := os.Getwd()
	if err != nil {
		return
	}

	pkgName := filepath.Base(sourceDir)
	fileName = fmt.Sprintf("%s_task.go", pkgName)
	outfile := filepath.Join(sourceDir, fileName)
	f, err := os.Create(outfile)
	if err != nil {
		return
	}
	defer f.Close()

	w := exampleWriter{Pkg: pkgName}
	err = w.Write(f)

	return
}
