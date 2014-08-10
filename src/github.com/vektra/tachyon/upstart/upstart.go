package upstart

import (
	"bytes"
	"fmt"
	"github.com/guelfey/go.dbus"
	"os/exec"
	"os/user"
	"strings"
)

type Conn struct {
	conn *dbus.Conn
}

type Job struct {
	u    *Conn
	path dbus.ObjectPath
}

const BusName = "com.ubuntu.Upstart"

func (u *Conn) object(path dbus.ObjectPath) *dbus.Object {
	return u.conn.Object(BusName, path)
}

func userAndHome() (string, string, error) {
	u, err := user.Current()
	if err != nil {
		out, nerr := exec.Command("sh", "-c", "getent passwd `id -u`").Output()

		if nerr != nil {
			return "", "", err
		}

		fields := bytes.Split(out, []byte(`:`))
		if len(fields) >= 6 {
			return string(fields[0]), string(fields[5]), nil
		}

		return "", "", fmt.Errorf("Unable to figure out the home dir")
	}

	return u.Username, u.HomeDir, nil
}

func Dial() (*Conn, error) {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return nil, err
	}

	user, home, err := userAndHome()
	if err != nil {
		return nil, err
	}

	methods := []dbus.Auth{dbus.AuthExternal(user), dbus.AuthCookieSha1(user, home)}
	if err = conn.Auth(methods); err != nil {
		conn.Close()
		return nil, err
	}

	if err = conn.Hello(); err != nil {
		conn.Close()
		conn = nil
	}

	return &Conn{conn}, nil
}

func (u *Conn) Close() error {
	return u.conn.Close()
}

func (u *Conn) Jobs() ([]*Job, error) {
	obj := u.object("/com/ubuntu/Upstart")

	var s []dbus.ObjectPath
	err := obj.Call("com.ubuntu.Upstart0_6.GetAllJobs", 0).Store(&s)
	if err != nil {
		return nil, err
	}

	var out []*Job

	for _, v := range s {
		out = append(out, &Job{u, v})
	}

	return out, nil
}

func (u *Conn) Job(name string) (*Job, error) {
	obj := u.object("/com/ubuntu/Upstart")

	var s dbus.ObjectPath
	err := obj.Call("com.ubuntu.Upstart0_6.GetJobByName", 0, name).Store(&s)
	if err != nil {
		return nil, err
	}

	return &Job{u, s}, nil
}

func (u *Conn) Instance(name string) (*Instance, error) {
	parts := strings.SplitN(name, "/", 2)

	job, err := u.Job(parts[0])
	if err != nil {
		return nil, err
	}

	inst := ""

	if len(parts) == 2 {
		inst = parts[1]
	}

	return job.Instance(inst)
}

func (u *Conn) EmitEvent(name string, env []string, wait bool) error {
	obj := u.object("/com/ubuntu/Upstart")
	return obj.Call("com.ubuntu.Upstart0_6.EmitEvent", 0, name, env, wait).Store()
}

type Instance struct {
	j    *Job
	path dbus.ObjectPath
}

func (j *Job) obj() *dbus.Object {
	return j.u.object(j.path)
}

func (i *Instance) obj() *dbus.Object {
	return i.j.u.object(i.path)
}

func (j *Job) Instances() ([]*Instance, error) {
	var instances []dbus.ObjectPath

	err := j.obj().Call("com.ubuntu.Upstart0_6.Job.GetAllInstances", 0).Store(&instances)
	if err != nil {
		return nil, err
	}

	var out []*Instance

	for _, inst := range instances {
		out = append(out, &Instance{j, inst})
	}

	return out, nil
}

func (j *Job) Instance(name string) (*Instance, error) {
	var path dbus.ObjectPath

	err := j.obj().Call("com.ubuntu.Upstart0_6.Job.GetInstanceByName", 0, name).Store(&path)
	if err != nil {
		return nil, err
	}

	return &Instance{j, path}, nil
}

func (j *Job) prop(name string) (string, error) {
	val, err := j.obj().GetProperty("com.ubuntu.Upstart0_6.Job." + name)
	if err != nil {
		return "", err
	}

	if str, ok := val.Value().(string); ok {
		return str, nil
	}

	return "", fmt.Errorf("Name was not a string")
}

func (j *Job) Name() (string, error) {
	return j.prop("name")
}

func (j *Job) Description() (string, error) {
	return j.prop("description")
}

func (j *Job) Author() (string, error) {
	return j.prop("author")
}

func (j *Job) Version() (string, error) {
	return j.prop("version")
}

func (j *Job) Pid() (int32, error) {
	insts, err := j.Instances()
	if err != nil {
		return 0, err
	}

	switch len(insts) {
	default:
		return 0, fmt.Errorf("More than 1 instances running, no single pid")
	case 0:
		return 0, fmt.Errorf("No instances of job available")
	case 1:
		procs, err := insts[0].Processes()
		if err != nil {
			return 0, err
		}

		switch len(procs) {
		default:
			return 0, fmt.Errorf("More than 1 processes running, no single pid")
		case 0:
			return 0, fmt.Errorf("No process running of any instances")
		case 1:
			return procs[0].Pid, nil
		}
	}
}

func (j *Job) Pids() ([]int32, error) {
	insts, err := j.Instances()
	if err != nil {
		return nil, err
	}

	var pids []int32

	for _, inst := range insts {
		procs, err := inst.Processes()
		if err != nil {
			return nil, err
		}

		for _, proc := range procs {
			pids = append(pids, proc.Pid)
		}
	}

	return pids, nil
}

func (j *Job) StartWithOptions(env []string, wait bool) (*Instance, error) {
	c := j.obj().Call("com.ubuntu.Upstart0_6.Job.Start", 0, env, wait)

	var path dbus.ObjectPath
	err := c.Store(&path)
	if err != nil {
		return nil, err
	}

	return &Instance{j, path}, nil
}

func (j *Job) Start() (*Instance, error) {
	return j.StartWithOptions([]string{}, true)
}

func (j *Job) StartAsync() (*Instance, error) {
	return j.StartWithOptions([]string{}, false)
}

func (j *Job) Restart() (*Instance, error) {
	wait := true
	c := j.obj().Call("com.ubuntu.Upstart0_6.Job.Restart", 0, []string{}, wait)

	var path dbus.ObjectPath
	err := c.Store(&path)
	if err != nil {
		return nil, err
	}

	return &Instance{j, path}, nil
}

func (j *Job) Stop() error {
	wait := true
	c := j.obj().Call("com.ubuntu.Upstart0_6.Job.Stop", 0, []string{}, wait)

	return c.Store()
}

func (i *Instance) strprop(name string) (string, error) {
	val, err := i.obj().GetProperty("com.ubuntu.Upstart0_6.Instance." + name)
	if err != nil {
		return "", err
	}

	if str, ok := val.Value().(string); ok {
		return str, nil
	}

	return "", fmt.Errorf("Name was not a string")
}

func (i *Instance) Name() (string, error) {
	return i.strprop("name")
}

func (i *Instance) Goal() (string, error) {
	return i.strprop("goal")
}

func (i *Instance) State() (string, error) {
	return i.strprop("state")
}

type Process struct {
	Name string
	Pid  int32
}

func (i *Instance) Processes() ([]Process, error) {
	val, err := i.obj().GetProperty("com.ubuntu.Upstart0_6.Instance.processes")

	if err != nil {
		return nil, err
	}

	var out []Process

	if ary, ok := val.Value().([][]interface{}); ok {
		for _, elem := range ary {
			out = append(out, Process{elem[0].(string), elem[1].(int32)})
		}
	} else {
		return nil, fmt.Errorf("Unable to decode processes property")
	}

	return out, nil
}

func (i *Instance) Pid() (int32, error) {
	processes, err := i.Processes()
	if err != nil {
		return 0, err
	}

	switch len(processes) {
	case 0:
		return 0, fmt.Errorf("No running processes for this instance")
	case 1:
		return processes[0].Pid, nil
	default:
		return 0, fmt.Errorf("More than one process for this instance")
	}
}

func (i *Instance) Start() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Start", 0, true)

	return c.Store()
}

func (i *Instance) StartAsync() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Start", 0, false)

	return c.Store()
}
func (i *Instance) Restart() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Restart", 0, true)

	return c.Store()
}

func (i *Instance) RestartAsync() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Restart", 0, false)

	return c.Store()
}

func (i *Instance) Stop() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Stop", 0, true)

	return c.Store()
}

func (i *Instance) StopAsync() error {
	c := i.obj().Call("com.ubuntu.Upstart0_6.Instance.Stop", 0, false)

	return c.Store()
}
