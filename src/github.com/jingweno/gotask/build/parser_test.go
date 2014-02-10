package build

import (
	"github.com/bmizerany/assert"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestParser_findImportPath(t *testing.T) {
	dir, _ := expandPath("../examples")
	importPath, err := findImportPath(dir)

	assert.Equal(t, nil, err)
	assert.Equal(t, "github.com/jingweno/gotask/examples", importPath)
}

func TestParser_Parse(t *testing.T) {
	p := NewParser()
	ts, err := p.Parse("../examples")

	assert.Equal(t, nil, err)
	assert.Equal(t, "examples", ts.Name)
	assert.Tf(t, strings.HasSuffix(ts.Dir, filepath.Join("github.com", "jingweno", "gotask", "examples")), "%s", ts.Dir)
	assert.Tf(t, strings.HasSuffix(ts.PkgObj, filepath.Join("github.com", "jingweno", "gotask", "examples.a")), "%s", ts.PkgObj)
	assert.Equal(t, "github.com/jingweno/gotask/examples", ts.ImportPath)
	assert.Equal(t, 2, len(ts.Tasks))

	// no task files found
	temp, err := ioutil.TempDir("", "go-task")
	assert.Equal(t, nil, err)

	ts, err = p.Parse(temp)
	assert.Equal(t, nil, err)
	assert.Equal(t, filepath.Base(temp), ts.Name)
	assert.Equal(t, ".", ts.ImportPath)
	assert.Equal(t, "", ts.PkgObj)
	assert.Equal(t, temp, ts.Dir)
	assert.Equal(t, 0, len(ts.Tasks))
}

func TestTaskParser_filterTaskFiles(t *testing.T) {
	files := []string{"file.go", "file_task.go", "task.go"}
	taskFiles := filterTaskFiles(files)

	assert.Equal(t, 1, len(taskFiles))
	assert.Equal(t, "file_task.go", taskFiles[0])
}

func TestParser_parseTasks(t *testing.T) {
	tasks, _ := parseTasks("../examples/say_hello_task.go")

	assert.Equal(t, 1, len(tasks))
	assert.Equal(t, "TaskSayHello", tasks[0].ActionName)
	assert.Equal(t, "say-hello", tasks[0].Name)
	assert.Equal(t, "Say hello to current user", tasks[0].Usage)
	assert.Equal(t, "Print out hello to current user", tasks[0].Description)
}
