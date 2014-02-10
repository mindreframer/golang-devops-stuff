package build

import (
	"io/ioutil"
	"os"
)

func Run(sourceDir string, args []string, isDebug bool) (err error) {
	parser := NewParser()
	taskSet, err := parser.Parse(sourceDir)
	if err != nil {
		return
	}

	err = withTempDir(isDebug, func(work string) (err error) {
		compiler := compiler{sourceDir: sourceDir, workDir: work, TaskSet: taskSet, isDebug: isDebug}
		execFile, err := compiler.Compile("")
		if err != nil {
			return
		}

		runner := runner{execFile}
		err = runner.Run(args)
		return
	})

	return
}

func Compile(sourceDir string, outfile string, isDebug bool) (err error) {
	parser := NewParser()
	taskSet, err := parser.Parse(sourceDir)
	if err != nil {
		return
	}

	err = withTempDir(isDebug, func(work string) (err error) {
		compiler := compiler{sourceDir: sourceDir, workDir: work, TaskSet: taskSet, isDebug: isDebug}
		_, err = compiler.Compile(outfile)
		return
	})

	return
}

func withTempDir(isDebug bool, f func(string) error) (err error) {
	temp, err := ioutil.TempDir("", "go-task")
	if err != nil {
		return
	}
	defer func() {
		if isDebug {
			debugf("keeping work directory %s", temp)
		} else {
			os.RemoveAll(temp)
		}
	}()

	if isDebug {
		debugf("building tasks in %s\n", temp)
	}
	err = f(temp)
	return
}
