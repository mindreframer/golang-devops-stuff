package upstart

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestConfigGenerate(t *testing.T) {
	c := DaemonConfig("puma", "puma -c blah.cfg")

	b := c.Generate()

	exp := `# puma daemon

description "puma service"
start on runlevel [2345]
stop on runlevel [!2345]
expect daemon
respawn
console log
exec puma -c blah.cfg
`

	if string(b) != exp {
		t.Log(exp)
		t.Log(string(b))
		t.Fatal("Config did not generate properly")
	}
}

func TestConfigGenerateAllOptions(t *testing.T) {
	c := DaemonConfig("puma", "puma -c blah.cfg")
	c.Directory = "/tmp"
	c.Emits = []string{"fun-times"}
	c.Env["FOO"] = "bar"
	c.Instance = "$INDEX"
	c.KillSignal = []string{"SIGTERM"}
	c.KillTimeout = 30
	c.Limit = []string{"blah"}
	c.Manual = true
	c.Nice = 0
	c.OomScore = 0
	c.ReloadSignal = "SIGUSR2"
	c.SetGid = "staff"
	c.SetUid = "deploy"
	c.Umask = 044
	c.Usage = "puma options"
	c.Version = "1.0-beta"

	b := c.Generate()

	exp := `# puma daemon

description "puma service"
usage "puma options"
version "1.0-beta"
start on runlevel [2345]
stop on runlevel [!2345]
emits fun-times
instance $INDEX
expect daemon
respawn
limit blah
console log
chdir /tmp
env FOO="bar"
kill signal SIGTERM
kill timeout 30
reload signal SIGUSR2
manual
nice 0
oom score 0
setgid staff
setuid deploy
umask 044
exec puma -c blah.cfg
`

	if string(b) != exp {
		t.Log(exp)
		t.Log(string(b))
		t.Fatal("Config did not generate properly")
	}
}

func TestConfigGeneratePostPreExec(t *testing.T) {
	c := DaemonConfig("puma", "puma -c blah.cfg")
	c.PreStart.Exec = "foo3"
	c.PostStart.Exec = "foo1"
	c.PreStop.Exec = "foo4"
	c.PostStop.Exec = "foo2"

	b := c.Generate()

	exp := `# puma daemon

description "puma service"
start on runlevel [2345]
stop on runlevel [!2345]
expect daemon
respawn
console log
pre-start exec foo3
post-start exec foo1
pre-stop exec foo4
post-stop exec foo2
exec puma -c blah.cfg
`

	if string(b) != exp {
		t.Log(exp)
		t.Log(string(b))
		t.Fatal("Config did not generate properly")
	}
}

func TestConfigGeneratePostPreScript(t *testing.T) {
	c := DaemonConfig("puma", "puma -c blah.cfg")
	c.PreStart.Script = "foo3\nbar"
	c.PostStart.Script = "foo1\nbar"
	c.PreStop.Script = "foo4\nbar"
	c.PostStop.Script = "foo2\nbar"

	b := c.Generate()

	exp := `# puma daemon

description "puma service"
start on runlevel [2345]
stop on runlevel [!2345]
expect daemon
respawn
console log
pre-start script
  foo3
  bar
end script
post-start script
  foo1
  bar
end script
pre-stop script
  foo4
  bar
end script
post-stop script
  foo2
  bar
end script
exec puma -c blah.cfg
`

	if string(b) != exp {
		t.Log(exp)
		t.Log(string(b))
		t.Fatal("Config did not generate properly")
	}
}

func TestConfigGenerateTask(t *testing.T) {
	c := TaskConfig("warmup-db", "mysql --warm-up")

	b := c.Generate()

	exp := `# warmup-db task

description "warmup-db task"
start on runlevel [2345]
stop on runlevel [!2345]
task
console log
exec mysql --warm-up
`

	if string(b) != exp {
		t.Log(exp)
		t.Log(string(b))
		t.Fatal("Config did not generate properly")
	}
}

func TestInstallCommand(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "upstart-test")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(tmpdir)

	InitDir = tmpdir

	exp := []byte("stuff")

	err = InstallConfig("blah", exp)
	if err != nil {
		panic(err)
	}

	config, err := ioutil.ReadFile(tmpdir + "/blah.conf")
	if err != nil {
		panic(err)
	}

	if !bytes.Equal(exp, config) {
		t.Error("Did not write the config")
	}
}

func TestConfigInstall(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "upstart-test")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(tmpdir)

	InitDir = tmpdir

	c := DaemonConfig("puma", "puma -c blah.conf")

	exp := c.Generate()

	err = c.Install()
	if err != nil {
		panic(err)
	}

	config, err := ioutil.ReadFile(tmpdir + "/puma.conf")
	if err != nil {
		panic(err)
	}

	if !bytes.Equal(exp, config) {
		t.Error("Did not write the config")
	}
}

func TestConfigExists(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "upstart-test")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(tmpdir)

	InitDir = tmpdir

	c := DaemonConfig("puma", "puma -c blah.conf")

	c.Generate()

	err = c.Install()
	if err != nil {
		panic(err)
	}

	if !c.Exists() {
		t.Error("Didn't find the config already there")
	}
}
