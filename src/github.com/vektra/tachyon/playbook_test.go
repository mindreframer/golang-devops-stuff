package tachyon

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSimplePlaybook(t *testing.T) {
	env := NewEnv(NewNestedScope(nil), DefaultConfig)
	p, err := NewPlaybook(env, "test/playbook1.yml")

	if err != nil {
		panic(err)
	}

	if len(p.Plays) != 2 {
		t.Fatalf("Didn't load 2 playbooks, loaded: %d", len(p.Plays))
	}

	x := p.Plays[1]

	if x.Hosts != "all" {
		t.Errorf("Hosts not all: was %s", x.Hosts)
	}

	vars := x.Vars

	a, ok := vars.Get("answer")

	if !ok {
		t.Fatalf("No var 'answer'")
	}

	if a.Read() != "Wuh, I think so" {
		t.Errorf("Unable to decode string var: %#v", a)
	}

	a, ok = vars.Get("port")

	if !ok {
		t.Fatalf("No var 'port'")
	}

	if a.Read() != 5150 {
		t.Errorf("Unable to decode numeric var: %#v", a.Read())
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

func totalRuntime(results []RunResult) time.Duration {
	cur := time.Duration(0)

	for _, res := range results {
		cur += res.Runtime
	}

	return cur
}

func TestPlaybookFuturesRunInParallel(t *testing.T) {
	run, _, err := RunCapture("test/future.yml")
	if err != nil {
		t.Fatalf("Unable to load test/future.yml")
	}

	total := run.Runtime.Seconds()

	if total > 1.1 || total < 0.9 {
		t.Errorf("Futures did not run in parallel: %f", total)
	}
}

func TestPlaybookFuturesCanBeWaitedOn(t *testing.T) {
	run, _, err := RunCapture("test/future.yml")
	if err != nil {
		t.Fatalf("Unable to load test/future.yml")
	}

	total := run.Runtime.Seconds()

	if total > 1.1 || total < 0.9 {
		t.Errorf("Futures did not run in parallel: %f", total)
	}
}

func TestPlaybookTaskIncludes(t *testing.T) {
	res, _, err := RunCapture("test/inc_parent.yml")
	if err != nil {
		t.Fatalf("Unable to run test/inc_parent.yml")
	}

	if filepath.Base(res.Results[0].Task.File) != "inc_child.yml" {
		t.Fatalf("Did not include tasks from child")
	}
}

func TestPlaybookTaskIncludesCanHaveVars(t *testing.T) {
	res, _, err := RunCapture("test/inc_parent2.yml")
	if err != nil {
		t.Fatalf("Unable to run test/inc_parent2.yml: %s", err)
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "oscar" {
		t.Fatalf("A variable was not passed into the included file")
	}

	d = res.Results[1].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "ellen" {
		t.Fatalf("A variable was not passed into the included file")
	}

	d = res.Results[2].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "Los Angeles" {
		t.Fatalf("A variable was not passed into the included file")
	}
}

func TestPlaybookRoleTasksInclude(t *testing.T) {
	res, _, err := RunCapture("test/site1.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site1.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "in role" {
		t.Fatalf("Task did not run from role")
	}
}

func TestPlaybookRoleHandlersInclude(t *testing.T) {
	res, _, err := RunCapture("test/site1.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site1.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[1].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "in role handler" {
		t.Fatalf("Task did not run from role")
	}
}

func TestPlaybookRoleVarsInclude(t *testing.T) {
	res, _, err := RunCapture("test/site2.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site2.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "from role var" {
		t.Fatalf("Task did not run from role")
	}
}

func TestPlaybookRoleAcceptsVars(t *testing.T) {
	res, _, err := RunCapture("test/site3.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site3.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "from site3" {
		t.Fatalf("Task did not run from role")
	}
}

func TestPlaybookRoleAcceptsInlineVars(t *testing.T) {
	res, _, err := RunCapture("test/site4.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site4.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "from site4" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookRoleIncludesSeeRoleFiles(t *testing.T) {
	res, _, err := RunCapture("test/site5.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site5.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "in special" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookRoleFilesAreSeen(t *testing.T) {
	res, _, err := RunCapture("test/site6.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site6.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "in my script" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookRoleDependenciesAreInvoked(t *testing.T) {
	res, _, err := RunCapture("test/site7.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site7.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "role7" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookWithItems(t *testing.T) {
	res, _, err := RunCapture("test/items.yml")
	if err != nil {
		t.Fatalf("Unable to run test/items.yml: %s", err)
	}

	if len(res.Results) != 3 {
		t.Fatalf("tasks were not included from the role")
	}

	if v, ok := res.Results[0].Result.Get("stdout"); !ok || v.Read() != "a" {
		t.Fatal("first isnt 'a'")
	}

	if v, ok := res.Results[1].Result.Get("stdout"); !ok || v.Read() != "b" {
		t.Fatal("second isnt 'b'")
	}

	if v, ok := res.Results[2].Result.Get("stdout"); !ok || v.Read() != "c" {
		t.Fatal("third isnt 'c'")
	}

}

func TestPlaybookRoleModulesAreAvailable(t *testing.T) {
	res, _, err := RunCapture("test/site8.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site8.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "from module" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookRoleModulesCanUseYAMLArgs(t *testing.T) {
	res, _, err := RunCapture("test/site9.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site9.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "from module" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}

func TestPlaybookRoleSubTasks(t *testing.T) {
	res, _, err := RunCapture("test/site10.yml")
	if err != nil {
		t.Fatalf("Unable to run test/site10.yml: %s", err)
	}

	if len(res.Results) == 0 {
		t.Fatalf("tasks were not included from the role")
	}

	d := res.Results[0].Result

	if v, ok := d.Get("stdout"); !ok || v.Read() != "in get" {
		t.Fatalf("Task did not run from role: %#v", d)
	}
}
