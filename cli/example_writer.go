package cli

import (
	"io"
	"text/template"
)

type exampleWriter struct {
	Pkg string
}

func (w *exampleWriter) Write(wr io.Writer) (err error) {
	obj := struct {
		Pkg string
	}{
		Pkg: w.Pkg,
	}
	err = exampleTmpl.Execute(wr, obj)
	return
}

var exampleTmpl = template.Must(template.New("main").Parse(`package {{.Pkg}}

import (
	"github.com/jingweno/gotask/tasking"
)

// NAME
//    hello - Say hello world
//
// DESCRIPTION
//    Say hello world
//
// OPTIONS
//    --verbose, -v
//        run in verbose mode
func TaskHello(t *tasking.T) {
	t.Log("Hello world")
}`))
