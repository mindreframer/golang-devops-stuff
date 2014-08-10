package view
import (
	"bytes"
	"html/template"
	"github.com/rzab/amber"
)

type stringTemplate struct {
	*template.Template
}

var UsePercentTemplate  = mustTemplate("usepercent.html")
var TooltipableTemplate = mustTemplate("tooltipable.html")

func mustTemplate(filename string) stringTemplate {
	text, err := Asset(filename)
	if err != nil {
		panic(err)
	}
	return stringTemplate{template.Must(template.New(filename).Parse(string(text)))}
}

func(st stringTemplate) Execute(data interface{}) (template.HTML, error) {
	clone, err := st.Template.Clone()
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := clone.Execute(buf, data); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func Bincompile() *template.Template {
	t := template.New("templates.html") // root template, must not t.New("templates.html") later, causes redefinition of the template
	template.Must(t.Parse("Empty"))     // initial template in case we won't have any

	if filename := "index.html"; true {
		// for cascaded templates do `for filename := range AssetNames() // range over keys' instead of `if'

		text, err := Asset(filename)
		if err != nil {
			panic(err)
		}
		subt := t.New(filename)
		subt.Funcs(amber.FuncMap)
		template.Must(subt.Parse(string(text)))
	}
	return t
}
