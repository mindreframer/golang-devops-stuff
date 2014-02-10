package cli

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/jingweno/gotask/build"
	"os"
	"path/filepath"
)

var compileFlag = cli.BoolFlag{Name: "compile, c", Usage: "compile the task binary to pkg.task but do not run it"}

func compileTasks(isDebug bool) (err error) {
	sourceDir, err := os.Getwd()
	if err != nil {
		return
	}

	fileName := fmt.Sprintf("%s.task", filepath.Base(sourceDir))
	outfile := filepath.Join(sourceDir, fileName)

	err = build.Compile(sourceDir, outfile, isDebug)
	return
}
