package tachyon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/flynn/go-shlex"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func HomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		su := os.Getenv("SUDO_USER")

		var out []byte
		var nerr error

		if su != "" {
			out, nerr = exec.Command("sh", "-c", "getent passwd "+su).Output()
		} else {
			out, nerr = exec.Command("sh", "-c", "getent passwd `id -u`").Output()
		}

		if nerr != nil {
			return "", err
		}

		fields := bytes.Split(out, []byte(`:`))
		if len(fields) >= 6 {
			return string(fields[5]), nil
		}

		return "", fmt.Errorf("Unable to figure out the home dir")
	}

	return u.HomeDir, nil
}
func dbg(format string, args ...interface{}) {
	fmt.Printf("[DBG] "+format+"\n", args...)
}

func yamlFile(path string, v interface{}) error {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, v)
}

func mapToStruct(m map[string]interface{}, tag string, v interface{}) error {
	e := reflect.ValueOf(v).Elem()

	t := e.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		name := strings.ToLower(f.Name)
		required := false

		parts := strings.Split(f.Tag.Get(tag), ",")

		switch len(parts) {
		case 0:
			// nothing
		case 1:
			name = parts[0]
		case 2:
			name = parts[0]
			switch parts[1] {
			case "required":
				required = true
			default:
				return fmt.Errorf("Unsupported tag flag: %s", parts[1])
			}
		}

		if val, ok := m[name]; ok {
			e.Field(i).Set(reflect.ValueOf(val))
		} else if required {
			return fmt.Errorf("Missing value for %s", f.Name)
		}
	}

	return nil
}

func ParseSimpleMap(s Scope, args string) (Vars, error) {
	args, err := ExpandVars(s, args)

	if err != nil {
		return nil, err
	}

	sm := make(Vars)

	parts, err := shlex.Split(args)

	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		ec := strings.SplitN(part, "=", 2)

		if len(ec) == 2 {
			sm[ec[0]] = Any(inferString(ec[1]))
		} else {
			sm[part] = Any(true)
		}
	}

	return sm, nil
}

func split2(s, sep string) (string, string, bool) {
	parts := strings.SplitN(s, sep, 2)

	if len(parts) == 0 {
		return "", "", false
	} else if len(parts) == 1 {
		return parts[0], "", false
	} else {
		return parts[0], parts[1], true
	}
}

func inferString(s string) interface{} {
	switch strings.ToLower(s) {
	case "true", "yes":
		return true
	case "false", "no":
		return false
	}

	if i, err := strconv.ParseInt(s, 0, 0); err == nil {
		return i
	}

	return s
}

func indentedYAML(v interface{}, indent string) (string, error) {
	str, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(str), "\n")

	out := make([]string, len(lines))

	for idx, l := range lines {
		if l == "" {
			out[idx] = l
		} else {
			out[idx] = indent + l
		}
	}

	return strings.Join(out, "\n"), nil
}

func arrayVal(v interface{}, indent string) string {
	switch sv := v.(type) {
	case string:
		var out string

		if strings.Index(sv, "\n") != -1 {
			sub := strings.Split(sv, "\n")
			out = strings.Join(sub, "\n"+indent+" | ")
			return fmt.Sprintf("%s-\\\n%s   | %s", indent, indent, out)
		} else {
			return fmt.Sprintf("%s- \"%s\"", indent, sv)
		}
	case int, uint, int32, uint32, int64, uint64:
		return fmt.Sprintf("%s- %d", indent, sv)
	case bool:
		return fmt.Sprintf("%s- %t", indent, sv)
	case map[string]interface{}:
		mv := indentedMap(sv, indent+"  ")
		return fmt.Sprintf("%s-\n%s", indent, mv)
	}

	return fmt.Sprintf("%s- %v", indent, v)
}

func indentedMap(m map[string]interface{}, indent string) string {
	var keys []string

	for k, _ := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var lines []string

	for _, k := range keys {
		v := m[k]

		switch sv := v.(type) {
		case string:
			var out string

			if strings.Index(sv, "\n") != -1 {
				sub := strings.Split(sv, "\n")
				out = strings.Join(sub, "\n"+indent+" | ")
				lines = append(lines, fmt.Sprintf("%s%s:\n%s | %s",
					indent, k, indent, out))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s: \"%s\"", indent, k, sv))
			}
		case int, uint, int32, uint32, int64, uint64:
			lines = append(lines, fmt.Sprintf("%s%s: %d", indent, k, sv))
		case bool:
			lines = append(lines, fmt.Sprintf("%s%s: %t", indent, k, sv))
		case map[string]interface{}:
			mv := indentedMap(sv, indent+"  ")
			lines = append(lines, fmt.Sprintf("%s%s:\n%s", indent, k, mv))
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %v", indent, k, sv))
		}
	}

	return strings.Join(lines, "\n")
}

func indentedVars(m Vars, indent string) string {
	var keys []string

	for k, _ := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var lines []string

	for _, k := range keys {
		v := m[k]

		switch sv := v.Read().(type) {
		case string:
			var out string

			if strings.Index(sv, "\n") != -1 {
				sub := strings.Split(sv, "\n")
				out = strings.Join(sub, "\n"+indent+" | ")
				lines = append(lines, fmt.Sprintf("%s%s:\n%s | %s",
					indent, k, indent, out))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s: \"%s\"", indent, k, sv))
			}
		case int, uint, int32, uint32, int64, uint64:
			lines = append(lines, fmt.Sprintf("%s%s: %d", indent, k, sv))
		case bool:
			lines = append(lines, fmt.Sprintf("%s%s: %t", indent, k, sv))
		case map[string]interface{}:
			mv := indentedMap(sv, indent+"  ")
			lines = append(lines, fmt.Sprintf("%s%s:\n%s", indent, k, mv))
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %v", indent, k, sv))
		}
	}

	return strings.Join(lines, "\n")
}

func inlineMap(m map[string]interface{}) string {
	var keys []string

	for k, _ := range m {
		keys = append(keys, k)
	}

	// Minor special case. If there is only one key and it's
	// named "command", just return the value.
	if len(keys) == 1 && keys[0] == "command" {
		for _, v := range m {
			if sv, ok := v.(string); ok {
				return sv
			}
		}
	}

	sort.Strings(keys)

	var lines []string

	for _, k := range keys {
		v := m[k]

		switch sv := v.(type) {
		case string:
			lines = append(lines, fmt.Sprintf("%s=%s", k, strconv.Quote(sv)))
		case int, uint, int32, uint32, int64, uint64:
			lines = append(lines, fmt.Sprintf("%s=%d", k, sv))
		case bool:
			lines = append(lines, fmt.Sprintf("%s=%t", k, sv))
		case map[string]interface{}:
			lines = append(lines, fmt.Sprintf("%s=(%s)", k, inlineMap(sv)))
		default:
			lines = append(lines, fmt.Sprintf("%s=`%v`", k, sv))
		}
	}

	return strings.Join(lines, " ")
}

func inlineVars(m Vars) string {
	var keys []string

	for k, _ := range m {
		keys = append(keys, k)
	}

	// Minor special case. If there is only one key and it's
	// named "command", just return the value.
	if len(keys) == 1 && keys[0] == "command" {
		for _, v := range m {
			if sv, ok := v.Read().(string); ok {
				return sv
			}
		}
	}

	sort.Strings(keys)

	var lines []string

	for _, k := range keys {
		v := m[k]

		switch sv := v.Read().(type) {
		case string:
			lines = append(lines, fmt.Sprintf("%s=%s", k, strconv.Quote(sv)))
		case int, uint, int32, uint32, int64, uint64:
			lines = append(lines, fmt.Sprintf("%s=%d", k, sv))
		case bool:
			lines = append(lines, fmt.Sprintf("%s=%t", k, sv))
		case map[string]interface{}:
			lines = append(lines, fmt.Sprintf("%s=(%s)", k, inlineMap(sv)))
		default:
			lines = append(lines, fmt.Sprintf("%s=`%v`", k, sv))
		}
	}

	return strings.Join(lines, " ")
}

func fileExist(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !fi.IsDir()
}

func gmap(args ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	if len(args)%2 != 0 {
		panic(fmt.Sprintf("Specify an even number of args: %d", len(args)))
	}

	i := 0

	for i < len(args) {
		m[args[i].(string)] = args[i+1]
		i += 2
	}

	return m
}

func ijson(args ...interface{}) []byte {
	b, err := json.Marshal(gmap(args...))
	if err != nil {
		panic(err)
	}

	return b
}
