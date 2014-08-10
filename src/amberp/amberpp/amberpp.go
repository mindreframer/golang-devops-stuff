package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/rzab/amber"
)

type tree struct {
	name   string
	leaves []*tree // template/parse has nodes, we ought to have leaves
	parent *tree
	ranged bool
	keys   []string
	decl   string
}

func (top *tree) touch(words []string) {
	if len(words) == 0 {
		return
	}
	if nt := top.lookup(words[0]); nt != nil {
		nt.touch(words[1:])
		return
	}
	nt := &tree{name: words[0], parent: top}
	top.leaves = append(top.leaves, nt)
	nt.touch(words[1:])
}

func (top *tree) walk(words []string) *tree {
	if len(words) == 0 {
		return top
	}
	for _, leaf := range top.leaves {
		if leaf.name == words[0] {
			return leaf.walk(words[1:])
		}
	}
	return nil
}

func (top tree) lookup(name string) *tree {
	for _, leaf := range top.leaves {
		if name == leaf.name {
			return leaf
		}
	}
	return nil
}

func dotted(bottom *tree) string {
	if bottom == nil || bottom.name == "" {
		return ""
	}
	parent := dotted(bottom.parent)
	if parent == "" {
		return bottom.name
	}
	return parent + "." + bottom.name
}

func indent(top tree, level int) string {
	s := strings.Repeat(" ", level) + "[" + top.name + "]\n"
	level += 2
	for _, leaf := range top.leaves {
		s += indent(*leaf, level)
	}
	level -= 2
	return s
}

func (top tree) String() string {
	return indent(top, 0)
}

type hash map[string]interface{}

func curly(s string) string {
	if strings.HasSuffix(s, "HTML") {
		return "<span dangerouslySetInnerHTML={{__html: " + s + "}} />"
		return "{<span dangerouslySetInnerHTML={{__html: " + s + "}} />.props.children}"
	}
	return "{" + s + "}"
}

func mkmap(top tree, jscriptMode bool, level int) interface{} {
	if len(top.leaves) == 0 {
		return curly(dotted(&top))
	}
	h := make(hash)
	for _, leaf := range top.leaves {
		if leaf.ranged {
			if len(leaf.keys) != 0 {
				kv := make(map[string]string)
				for _, k := range leaf.keys {
					kv[k] = curly(leaf.decl + "." + k)
				}
				h[leaf.name] = []map[string]string{kv}
			} else {
				h[leaf.name] = []string{}
			}
		} else {
			h[leaf.name] = mkmap(*leaf, jscriptMode, level+1)
		}
	}
	if jscriptMode && level == 0 {
		h["CLASSNAME"] = "className"
	}
	return h
}

/* func string_hash(h interface{}) string {
	return hindent(h.(hash), 0)
}

func hindent(h hash, level int) string {
	s := ""
	for k, v := range h {
		s += strings.Repeat(" ", level) + "(" + k + ")\n"
		vv, ok := v.(hash)
		if ok && len(vv) > 0 {
			level += 2
			s += hindent(vv, level)
			level -= 2
		} else {
			s += strings.Repeat(" ", level + 2) + fmt.Sprint(v) + "\n"
		}
	}
	return s
} // */

type dotValue struct {
	s     string
	hashp *hash
}

func (dv dotValue) GoString() string {
	return dv.GoString()
}

func (dv dotValue) String() string {
	v := dv.s
	delete(*dv.hashp, "dot")
	return v
}

func dot(dot interface{}, key string) hash {
	h := dot.(hash)
	h["dot"] = dotValue{s: curly(key), hashp: &h}
	return h
}

var dotFuncs = map[string]interface{}{"dot": dot}

func main() {
	var (
		outputFile  string
		definesFile string
		prettyPrint bool
		jscriptMode bool
	)

	for _, name := range []string{"o", "output"} {
		flag.StringVar(&outputFile, name, "", "Output file")
	}
	for _, name := range []string{"d", "defines"} {
		flag.StringVar(&definesFile, name, "", "Defines file")
	}
	for _, name := range []string{"pp", "prettyprint"} {
		flag.BoolVar(&prettyPrint, name, false, "Pretty print")
	}
	for _, name := range []string{"j", "javascript"} {
		flag.BoolVar(&jscriptMode, name, false, "Javascript mode")
	}

	flag.Parse()
	inputFile := flag.Arg(0)

	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "No input file specified.")
		flag.Usage()
		os.Exit(2)
	}

	inputText := ""
	if definesFile != "" {
		b, err := ioutil.ReadFile(definesFile)
		check(err)
		inputText += compile(b, prettyPrint, jscriptMode)
		if inputText[len(inputText)-1] == '\n' { // amber does add this '\n', which is fine for the end of a file, which inputText is not
			inputText = inputText[:len(inputText)-1]
		}
	}
	b, err := ioutil.ReadFile(inputFile)
	check(err)
	inputText += compile(b, prettyPrint, jscriptMode)

	fstplate, err := template.New("fst").Funcs(dotFuncs).Delims("[[", "]]").Parse(inputText)
	check(err)
	fst := execute(fstplate, hash{})

	if !jscriptMode {
		writeFile(outputFile, fst)
		return
	}

	sndplate, err := template.New("snd").Funcs(template.FuncMap(amber.FuncMap)).Parse(fst)
	check(err)

	m := data(sndplate.Tree, jscriptMode) //; fmt.Printf("data => %+v\nstring_hash(data) => %+v", m, string_hash(m))
	snd := execute(sndplate, m)
	snd = regexp.MustCompile("</?script>").ReplaceAllLiteralString(snd, "")

	writeFile(outputFile, snd)
}

func writeFile(optFilename, s string) {
	b := []byte(s)
	if optFilename != "" {
		check(ioutil.WriteFile(optFilename, b, 0644))
	} else {
		os.Stdout.Write(b)
	}
}

func compile(input []byte, prettyPrint, jscriptMode bool) string {
	compiler := amber.New()
	compiler.PrettyPrint = prettyPrint // compiler.Options.PrettyPrint?
	if jscriptMode {
		compiler.ClassName = "className"
	}

	check(compiler.Parse(string(input)))
	s, err := compiler.CompileString()
	check(err)
	return s
}

func execute(emplate *template.Template, data interface{}) string {
	buf := new(bytes.Buffer)
	check(emplate.Execute(buf, data))
	return buf.String()
}

func data(TREE *parse.Tree, jscriptMode bool) interface{} {
	if TREE == nil || TREE.Root == nil {
		return "{}" // mkmap(tree{})
	}

	data := tree{}
	vars := map[string][]string{}

	for _, node := range TREE.Root.Nodes { // here we go
		switch node.Type() {
		case parse.NodeAction:
			actionNode := node.(*parse.ActionNode)
			decl := actionNode.Pipe.Decl

			for _, cmd := range actionNode.Pipe.Cmds {
				if cmd.NodeType != parse.NodeCommand {
					continue
				}
				for _, arg := range cmd.Args {
					var ident []string
					switch arg.Type() {

					case parse.NodeField:
						ident = arg.(*parse.FieldNode).Ident

						if len(decl) > 0 && len(decl[0].Ident) > 0 {
							vars[decl[0].Ident[0]] = ident
						}
						data.touch(ident)

					case parse.NodeVariable:
						ident = arg.(*parse.VariableNode).Ident

						if words, ok := vars[ident[0]]; ok {
							words := append(words, ident[1:]...)
							data.touch(words)
							if len(decl) > 0 && len(decl[0].Ident) > 0 {
								vars[decl[0].Ident[0]] = words
							}
						}
					}
				}
			}
		case parse.NodeRange:
			rangeNode := node.(*parse.RangeNode)
			decl := rangeNode.Pipe.Decl[len(rangeNode.Pipe.Decl)-1].String()
			keys := []string{}

			for _, ifnode := range rangeNode.List.Nodes {
				switch ifnode.Type() {
				case parse.NodeAction:
					keys = append(keys, getKeys(decl, ifnode)...)
				case parse.NodeIf:
					for _, z := range ifnode.(*parse.IfNode).List.Nodes {
						if z.Type() == parse.NodeAction {
							keys = append(keys, getKeys(decl, z)...)
						}
					}
				}
			}

			// fml
			arg0 := rangeNode.Pipe.Cmds[0].Args[0].String()
			if words, ok := vars[arg0]; ok {
				if leaf := data.walk(words); leaf != nil {
					leaf.ranged = true
					leaf.keys = append(leaf.keys, keys...)
					leaf.decl = decl // redefined $
				}
			}
		}
	}
	return mkmap(data, jscriptMode, 0)
}

func getKeys(decl string, parseNode parse.Node) (keys []string) {
	for _, cmd := range parseNode.(*parse.ActionNode).Pipe.Cmds {
		if cmd.NodeType != parse.NodeCommand {
			continue
		}
		for _, arg := range cmd.Args {
			if arg.Type() != parse.NodeVariable {
				continue
			}
			ident := arg.(*parse.VariableNode).Ident
			if len(ident) < 2 || ident[0] != decl {
				continue
			}
			keys = append(keys, ident[1])
		}
	}
	return
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
