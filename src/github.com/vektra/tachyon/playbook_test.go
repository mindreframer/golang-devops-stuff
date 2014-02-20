package tachyon

import (
	"testing"
)

func TestSimplePlaybook(t *testing.T) {
	p, err := LoadPlaybook("test/playbook1.yml")

	if err != nil {
		panic(err)
	}

	if len(p) != 1 {
		t.Fatalf("Didn't load 1 playbook, loaded: %d", len(p))
	}

	x := p[0]

	if x.Hosts != "all" {
		t.Errorf("Hosts not all: was %s", x.Hosts)
	}

	vars := x.Vars

	if vars["answer"] != "Wuh, I think so" {
		t.Errorf("Unable to decode string var: %#v", vars["answer"])
	}

	if vars["port"] != 5150 {
		t.Errorf("Unable to decode numeric var")
	}

	if len(x.VarsFiles) != 2 {
		t.Fatalf("Unable to decode varsfiles, got %d", len(x.VarsFiles))
	}

	f := x.VarsFiles[0]

	if f != "common_vars.yml" {
		t.Errorf("Unable to decode literal vars_files")
	}

	f2 := x.VarsFiles[1].([]interface{})

	if f2[1].(string) != "default_os.yml" {
		t.Errorf("Unable to decode list vars_files")
	}

	tasks := x.Tasks

	if len(tasks) < 5 {
		t.Errorf("Failed to decode the proper number of tasks: %d", len(tasks))
	}

	if tasks[3].Args() != "echo {{port}}" {
		t.Errorf("Failed to decode templating in action: %#v", tasks[3].Args())
	}
}
