package upstart

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var InitDir = "/etc/init"

type Script string

func (s Script) Indented() string {
	return "  " + strings.Join(strings.Split(string(s), "\n"), "\n  ")
}

type Code struct {
	Exec   string
	Script Script
}

func (c Code) Set() bool {
	return c.Exec != "" || c.Script != ""
}

func (c Code) Output(name string) string {
	if c.Exec != "" {
		return fmt.Sprintf("%s exec %s\n", name, c.Exec)
	}

	if c.Script != "" {
		return fmt.Sprintf("%s script\n%s\nend script\n", name, c.Script.Indented())
	}

	return ""
}

type Config struct {
	Name string
	Type string

	Console     string
	Directory   string
	Description string
	Emits       []string
	Env         map[string]string
	Exec        string
	Expect      string
	Instance    string
	KillSignal  []string
	KillTimeout int
	Limit       []string
	Manual      bool
	Nice        int
	OomScore    int

	PostStart Code
	PostStop  Code
	PreStart  Code
	PreStop   Code

	ReloadSignal string
	Respawn      bool
	Script       Script
	SetGid       string
	SetUid       string
	StartOn      string
	StopOn       string

	Umask   int
	Usage   string
	Version string
}

func NewConfig() *Config {
	return &Config{
		Nice:     -1,
		OomScore: -1000,
		Umask:    -1,
		Env:      make(map[string]string),
	}
}

func DaemonConfig(name string, cmd string) *Config {
	cfg := &Config{
		Nice:     -1,
		OomScore: -1000,
		Umask:    -1,
		Env:      make(map[string]string),

		Name:        name,
		Type:        "daemon",
		Console:     "log",
		Description: fmt.Sprintf("%s service", name),
		Exec:        cmd,
		Expect:      "daemon",
		Respawn:     true,
		StartOn:     "runlevel [2345]",
		StopOn:      "runlevel [!2345]",
	}

	return cfg
}

func TaskConfig(name string, cmd string) *Config {
	cfg := &Config{
		Nice:     -1,
		OomScore: -1000,
		Umask:    -1,
		Env:      make(map[string]string),

		Name:        name,
		Type:        "task",
		Console:     "log",
		Description: fmt.Sprintf("%s task", name),
		Exec:        cmd,
		StartOn:     "runlevel [2345]",
		StopOn:      "runlevel [!2345]",
	}

	return cfg
}

func (c *Config) UpdateDefaults() {
	if c.Description == "" {
		c.Description = fmt.Sprintf("%s %s", c.Name, c.Type)
	}
}

func (c *Config) Foreground() {
	c.Expect = ""
}

func (c *Config) Generate() []byte {
	var buf bytes.Buffer

	c.UpdateDefaults()

	buf.WriteString(fmt.Sprintf("# %s %s\n\n", c.Name, c.Type))

	buf.WriteString(fmt.Sprintf("description \"%s\"\n", c.Description))

	if c.Usage != "" {
		buf.WriteString(fmt.Sprintf("usage \"%s\"\n", c.Usage))
	}

	if c.Version != "" {
		buf.WriteString(fmt.Sprintf("version \"%s\"\n", c.Version))
	}

	buf.WriteString(fmt.Sprintf("start on %s\n", c.StartOn))
	buf.WriteString(fmt.Sprintf("stop on %s\n", c.StopOn))

	if c.Type == "task" {
		buf.WriteString("task\n")
	}

	for _, e := range c.Emits {
		buf.WriteString(fmt.Sprintf("emits %s\n", e))
	}

	if c.Instance != "" {
		buf.WriteString(fmt.Sprintf("instance %s\n", c.Instance))
	}

	if c.Expect != "" {
		buf.WriteString(fmt.Sprintf("expect %s\n", c.Expect))
	}

	if c.Respawn {
		buf.WriteString("respawn\n")
	}

	if len(c.Limit) > 0 {
		ls := strings.Join(c.Limit, " ")
		buf.WriteString(fmt.Sprintf("limit %s\n", ls))
	}

	if c.Console != "" {
		buf.WriteString(fmt.Sprintf("console %s\n", c.Console))
	}

	if c.Directory != "" {
		buf.WriteString(fmt.Sprintf("chdir %s\n", c.Directory))
	}

	for k, v := range c.Env {
		buf.WriteString(fmt.Sprintf("env %s=\"%s\"\n", k, v))
	}

	for _, v := range c.KillSignal {
		buf.WriteString(fmt.Sprintf("kill signal %s\n", v))
	}

	if c.KillTimeout != 0 {
		buf.WriteString(fmt.Sprintf("kill timeout %d\n", c.KillTimeout))
	}

	if c.ReloadSignal != "" {
		buf.WriteString(fmt.Sprintf("reload signal %s\n", c.ReloadSignal))
	}

	if c.Manual {
		buf.WriteString("manual\n")
	}

	if c.Nice != -1 {
		buf.WriteString(fmt.Sprintf("nice %d\n", c.Nice))
	}

	if c.OomScore != -1000 {
		buf.WriteString(fmt.Sprintf("oom score %d\n", c.OomScore))
	}

	if c.SetGid != "" {
		buf.WriteString(fmt.Sprintf("setgid %s\n", c.SetGid))
	}

	if c.SetUid != "" {
		buf.WriteString(fmt.Sprintf("setuid %s\n", c.SetUid))
	}

	if c.Umask != -1 {
		buf.WriteString(fmt.Sprintf("umask %03o\n", c.Umask))
	}

	if c.PreStart.Set() {
		buf.WriteString(c.PreStart.Output("pre-start"))
	}

	if c.PostStart.Set() {
		buf.WriteString(c.PostStart.Output("post-start"))
	}

	if c.PreStop.Set() {
		buf.WriteString(c.PreStop.Output("pre-stop"))
	}

	if c.PostStop.Set() {
		buf.WriteString(c.PostStop.Output("post-stop"))
	}

	if c.Script != "" {
		s := fmt.Sprintf("script\n%s\nend script\n", c.Script.Indented())
		buf.WriteString(s)
	}

	if c.Exec != "" {
		buf.WriteString(fmt.Sprintf("exec %s\n", c.Exec))
	}

	return buf.Bytes()
}

func (c *Config) Install() error {
	return InstallConfig(c.Name, c.Generate())
}

func (c *Config) Exists() bool {
	_, err := os.Stat(filepath.Join(InitDir, c.Name+".conf"))
	if err == nil {
		return true
	}

	return false
}

func InstallConfig(name string, config []byte) error {
	path := filepath.Join(InitDir, name+".conf")
	return ioutil.WriteFile(path, config, 0644)
}
