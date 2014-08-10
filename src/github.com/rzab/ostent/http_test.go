package ostent
import (
	"testing"
	"bytes"
	"reflect"
	"html/template"
	"github.com/rzab/amber"
)

// Traverses the type, fails on any pointer field
// Intended to reveal the pointers, they're not comparable in templates (see amber/runtime.go)
// Ruled to use json() in the templates to compare values
func test_pagedatatype(t *testing.T, typ reflect.Type) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		ftyp := field.Type
		kind := ftyp.Kind()
		if kind == reflect.Ptr {
			t.Errorf("%s.%s\tis a pointer", typ.Name(), field.Name)
		}
		if kind == reflect.Struct {
			test_pagedatatype(t, ftyp)
		}
	}
}

func Test_pageDataType(t *testing.T) {
	return // disabled for now // disabled henceforth
	test_pagedatatype(t, reflect.TypeOf(PageData{}))
}

func execute_template(IN string, DATA interface{}) (string, error) {
	buf := new(bytes.Buffer)
	if err := template.Must(template.New("tpl").Funcs(amber.FuncMap).Parse(IN)).Execute(buf, DATA); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Test_templatecomparison(t *testing.T) {
	type that struct {
		That *bool
	}
	newbool := func(value bool) *bool {
		b := new(bool)
		*b = value
		return b
	}
	for i, v := range []struct{
		in string
		data interface{}
		cmp string
	}{
		{`{{if .This }}That{{end}}`, struct{This string}{""},  ""},
		{`{{if .This }}That{{end}}`, struct{This string}{"a"}, "That"},
		{`{{if .This }}That{{end}}`, struct{This *bool}{}, ""},
		{`{{if .This }}Thta{{end}}`, struct{This *bool}{newbool(true)}, "Thta"},
//		{`{{if .This }}That{{end}}`, struct{This *bool}{newbool(false)}, ""}, // should fail

		{`{{not .This}}`, struct{This *bool}{newbool(true),}, "false"},
		{`{{not .This}}`, struct{This bool}{false,}, "true"},
// 		{`{{not .This}}`, struct{This *bool}{newbool(false),}, "true"}, // should fail

		{`{{and .This}}`, struct{This  bool}{true}, "true"},
		{`{{and .This}}`, struct{This  bool}{false}, "false"},
		{`{{and .This}}`, struct{This *bool}{newbool(true),}, "true"},
		{`{{and .This}}`, struct{This *bool}{newbool(false),}, "false"},

		{`{{and .This .That}}`, struct{This *bool; That *bool}{newbool(true),newbool(true),}, "true"},
		{`{{and .This .That}}`, struct{This *bool; That *bool}{newbool(true),newbool(false),}, "false"},
		{`{{and .This .That}}`, struct{This *bool; That *bool}{newbool(false),newbool(false),}, "false"},

		{`{{json .This}}`, struct{This bool}{true}, "true"},
		{`{{json .This}}`, struct{This bool}{false}, "false"},
		{`{{json .This}}`, struct{This bool}{},      "false"},
		{`{{json .This.That}}`, struct{This that}{that{newbool(true)}},  "true"},
		{`{{json .This.That}}`, struct{This that}{that{newbool(false)}}, "false"},
		{`{{json .This.That}}`, struct{This that}{that{}},               "null"}, // NB
	} {
		cmp, err := execute_template(v.in, v.data)
		if err != nil {
			t.Error(err)
		}
		if cmp != v.cmp {
			t.Errorf("[%d] Mismatch: execute_template(, %+v) == %v != %v\n", i, v.data, v.cmp, cmp)
		}
	}
}
