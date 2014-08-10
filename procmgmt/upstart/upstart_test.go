package upstart

import (
	"bytes"
	"github.com/vektra/tachyon"
	us "github.com/vektra/tachyon/upstart"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

var jobName string = "atd"
var runUpstartTests = false

func init() {
	if s := os.Getenv("TEST_JOB"); s != "" {
		jobName = s
	}

	c := exec.Command("which", "initctl")
	c.Run()
	runUpstartTests = c.ProcessState.Success()
}

func TestInstall(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	dest := "/etc/init/upstart-test-daemon.conf"

	defer os.Remove(dest)

	opts := `name=upstart-test-daemon file=test-daemon.conf.sample`

	res, err := tachyon.RunAdhocTask("upstart/install", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	_, err = os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDaemon(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	defer os.Remove("/etc/init/upstart-test-daemon.conf")

	opts := `name=upstart-test-daemon command="date"`

	res, err := tachyon.RunAdhocTask("upstart/daemon", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}
}

func TestDaemonScripts(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	dest := "/etc/init/upstart-test-daemon.conf"

	defer os.Remove(dest)

	opts := `name=upstart-test-daemon command="date" pre_start=@prestart.sample post_start=@poststart.sample pre_stop=@prestop.sample post_stop=@poststop.sample`

	res, err := tachyon.RunAdhocTask("upstart/daemon", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	body, err := ioutil.ReadFile(dest)
	if err != nil {
		panic(err)
	}

	idx := bytes.Index(body, []byte("this is a prestart sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a poststart sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a prestop sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a poststop sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}
}

func TestTaskScripts(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	dest := "/etc/init/upstart-test-daemon.conf"

	defer os.Remove(dest)

	opts := `name=upstart-test-daemon command="date" pre_start=@prestart.sample post_start=@poststart.sample pre_stop=@prestop.sample post_stop=@poststop.sample`

	res, err := tachyon.RunAdhocTask("upstart/task", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	body, err := ioutil.ReadFile(dest)
	if err != nil {
		panic(err)
	}

	idx := bytes.Index(body, []byte("this is a prestart sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a poststart sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a prestop sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}

	idx = bytes.Index(body, []byte("this is a poststop sample script"))
	if idx == -1 {
		t.Error("config didn't contain our script")
	}
}

func TestTask(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	defer os.Remove("/etc/init/upstart-test-task.conf")

	opts := `name=upstart-test-task command="date"`

	res, err := tachyon.RunAdhocTask("upstart/task", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}
}

func TestRestart(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	opts := "name=" + jobName

	u, err := us.Dial()
	if err != nil {
		panic(err)
	}

	j, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	prev, err := j.Pid()
	if err != nil {
		panic(err)
	}

	res, err := tachyon.RunAdhocTask("upstart/restart", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	cur, err := j.Pid()
	if err != nil {
		panic(err)
	}

	if res.Data.Get("pid") != cur {
		t.Log(res.Data)
		t.Fatal("pid not set properly")
	}

	if prev == cur {
		t.Fatal("restart did not happen")
	}
}

func TestStop(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	opts := "name=" + jobName

	u, err := us.Dial()
	if err != nil {
		panic(err)
	}

	j, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	defer j.Start()

	res, err := tachyon.RunAdhocTask("upstart/stop", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	res, err = tachyon.RunAdhocTask("upstart/stop", opts)
	if err != nil {
		panic(err)
	}

	if res.Changed {
		t.Fatal("change detected improperly")
	}
}

func TestStart(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	opts := "name=" + jobName

	u, err := us.Dial()
	if err != nil {
		panic(err)
	}

	j, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	defer j.Start()

	err = j.Stop()
	if err != nil {
		panic(err)
	}

	res, err := tachyon.RunAdhocTask("upstart/start", opts)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	act, ok := res.Get("pid")
	if !ok {
		t.Fatal("pid not set")
	}

	pid, err := j.Pid()
	if err != nil {
		panic(err)
	}

	if pid != act.Read() {
		t.Fatal("job did not start?")
	}

	res, err = tachyon.RunAdhocTask("upstart/start", opts)
	if err != nil {
		panic(err)
	}

	if res.Changed {
		t.Fatal("change detected improperly")
	}

}

func TestStartWithEnv(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	opts := "name=" + jobName

	u, err := us.Dial()
	if err != nil {
		panic(err)
	}

	j, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	defer j.Start()

	err = j.Stop()
	if err != nil {
		panic(err)
	}

	td := tachyon.TaskData{
		"upstart/start": map[interface{}]interface{}{
			"name": jobName,
			"env": map[interface{}]interface{}{
				"BAR": "foo",
			},
		},
	}

	res, err := tachyon.RunAdhocTaskVars(td)
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("no change detected")
	}

	act, ok := res.Get("pid")
	if !ok {
		t.Fatal("pid not set")
	}

	pid, err := j.Pid()
	if err != nil {
		panic(err)
	}

	if pid != act.Read() {
		t.Fatal("job did not start?")
	}

	res, err = tachyon.RunAdhocTask("upstart/start", opts)
	if err != nil {
		panic(err)
	}

	if res.Changed {
		t.Fatal("change detected improperly")
	}

}
