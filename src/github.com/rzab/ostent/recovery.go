package ostent
import (
	"runtime"
	"net/http"
	"html/template"
)

type Recovery bool // true stands for production

func(RC Recovery) ConstructorFunc(hf http.HandlerFunc) http.Handler {
	return RC.Constructor(http.HandlerFunc(hf))
}

func(RC Recovery) Constructor(HANDLER http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// TODO panic(err); return for hijacked connection
				w.WriteHeader(panicstatuscode) // NB

				var description string
				if err, ok := err.(error); ok {
					description = err.Error()
				}
				var stack string
				if !RC { // if !production
					sbuf := make([]byte, 4096 - len(panicstatustext) - len(description))
					size := runtime.Stack(sbuf, false)
					stack = string(sbuf[:size])
				}
				if tpl, err := rctemplate.Clone(); err == nil { // otherwise bail out
					tpl.Execute(w, struct {
						Title, Description, Stack string
					}{
						Title:       panicstatustext,
						Description: description,
						Stack:       stack,
					})
				}
			}
		}()
		HANDLER.ServeHTTP(w, r)
	})
}

const panicstatuscode = http.StatusInternalServerError
var   panicstatustext = statusLine(panicstatuscode)

var rctemplate = template.Must(template.New("recovery.html").Parse(`
<html>
<head><title>{{.Title}}</title></head>
<body bgcolor="white">
<center><h1>{{.Description}}</h1></center>
<hr><pre>{{.Stack}}</pre>
</body>
</html>
`))
