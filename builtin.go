package tachyon

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/flynn/go-shlex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

func captureCmd(c *exec.Cmd, show bool) ([]byte, []byte, error) {
	stdout, err := c.StdoutPipe()

	if err != nil {
		return nil, nil, err
	}

	defer stdout.Close()

	var wg sync.WaitGroup

	var bout bytes.Buffer
	var berr bytes.Buffer

	prefix := []byte(`| `)

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := bufio.NewReader(stdout)

		for {
			line, err := buf.ReadSlice('\n')

			if err != nil {
				break
			}

			bout.Write(line)

			if show {
				os.Stdout.Write(prefix)
				os.Stdout.Write(line)
			}
		}
	}()

	stderr, err := c.StderrPipe()

	if err != nil {
		stdout.Close()
		return nil, nil, err
	}

	defer stderr.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := bufio.NewReader(stderr)

		for {
			line, err := buf.ReadSlice('\n')

			if err != nil {
				break
			}

			berr.Write(line)

			if show {
				os.Stdout.Write(prefix)
				os.Stdout.Write(line)
			}
		}
	}()

	c.Start()

	wg.Wait()

	err = c.Wait()

	return bout.Bytes(), berr.Bytes(), err
}

type CommandResult struct {
	ReturnCode int
	Stdout     []byte
	Stderr     []byte
}

func RunCommand(env *CommandEnv, parts ...string) (*CommandResult, error) {
	c := exec.Command(parts[0], parts[1:]...)

	if env.Env.config.ShowCommandOutput {
		fmt.Printf("RUN: %s\n", strings.Join(parts, " "))
	}

	rc := 0

	stdout, stderr, err := captureCmd(c, env.Env.config.ShowCommandOutput)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			rc = 1
		} else {
			return nil, err
		}
	}

	return &CommandResult{rc, stdout, stderr}, nil
}

func RunCommandInEnv(env *CommandEnv, unixEnv []string, parts ...string) (*CommandResult, error) {
	c := exec.Command(parts[0], parts[1:]...)
	c.Env = unixEnv

	if env.Env.config.ShowCommandOutput {
		fmt.Printf("RUN: %s\n", strings.Join(parts, " "))
	}

	rc := 0

	stdout, stderr, err := captureCmd(c, env.Env.config.ShowCommandOutput)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			rc = 1
		} else {
			return nil, err
		}
	}

	return &CommandResult{rc, stdout, stderr}, nil
}

func runCmd(env *CommandEnv, ignore bool, parts ...string) (*Result, error) {
	cmd, err := RunCommand(env, parts...)
	if !ignore && err != nil {
		return nil, err
	}

	r := NewResult(true)

	r.Add("rc", cmd.ReturnCode)
	r.Add("stdout", strings.TrimSpace(string(cmd.Stdout)))
	r.Add("stderr", strings.TrimSpace(string(cmd.Stderr)))

	if str, ok := renderShellResult(r); ok {
		r.Add("_result", str)
	}

	return r, nil
}

type CommandCmd struct {
	Command    string `tachyon:"command,required"`
	Creates    string `tachyon:"creates"`
	IgnoreFail bool   `tachyon:"ignore_failure"`
}

func (cmd *CommandCmd) Run(env *CommandEnv) (*Result, error) {
	if cmd.Creates != "" {
		if _, err := os.Stat(cmd.Creates); err == nil {
			r := NewResult(false)
			r.Add("rc", 0)
			r.Add("exists", cmd.Creates)

			return r, nil
		}
	}

	parts, err := shlex.Split(cmd.Command)

	if err != nil {
		return nil, err
	}

	return runCmd(env, cmd.IgnoreFail, parts...)
}

func (cmd *CommandCmd) ParseArgs(s Scope, args string) (Vars, error) {
	if args == "" {
		return Vars{}, nil
	}

	return Vars{"command": Any(args)}, nil
}

type ShellCmd struct {
	Command    string `tachyon:"command,required"`
	Creates    string `tachyon:"creates"`
	IgnoreFail bool   `tachyon:"ignore_failure"`
}

func (cmd *ShellCmd) Run(env *CommandEnv) (*Result, error) {
	if cmd.Creates != "" {
		if _, err := os.Stat(cmd.Creates); err == nil {
			r := NewResult(false)
			r.Add("rc", 0)
			r.Add("exists", cmd.Creates)

			return r, nil
		}
	}

	return runCmd(env, cmd.IgnoreFail, "sh", "-c", cmd.Command)
}

func (cmd *ShellCmd) ParseArgs(s Scope, args string) (Vars, error) {
	if args == "" {
		return Vars{}, nil
	}

	return Vars{"command": Any(args)}, nil
}

func renderShellResult(res *Result) (string, bool) {
	rcv, ok := res.Get("rc")
	if !ok {
		return "", false
	}

	stdoutv, ok := res.Get("stdout")
	if !ok {
		return "", false
	}

	stderrv, ok := res.Get("stderr")
	if !ok {
		return "", false
	}

	rc := rcv.Read().(int)
	stdout := stdoutv.Read().(string)
	stderr := stderrv.Read().(string)

	if rc == 0 && len(stdout) == 0 && len(stderr) == 0 {
		return "", true
	} else if len(stderr) == 0 && len(stdout) < 60 {
		stdout = strings.Replace(stdout, "\n", " ", -1)
		return fmt.Sprintf(`rc: %d, stdout: "%s"`, rc, stdout), true
	}

	return "", false
}

type CopyCmd struct {
	Src  string `tachyon:"src,required"`
	Dest string `tachyon:"dest,required"`
}

func md5file(path string) ([]byte, error) {
	h := md5.New()

	i, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(h, i); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (cmd *CopyCmd) Run(env *CommandEnv) (*Result, error) {
	var src string

	if cmd.Src[0] == '/' {
		src = cmd.Src
	} else {
		src = env.Paths.File(cmd.Src)
	}

	input, err := os.Open(src)

	if err != nil {
		return nil, err
	}

	srcStat, err := os.Stat(src)
	if err != nil {
		return nil, err
	}

	srcDigest, err := md5file(src)
	if err != nil {
		return nil, err
	}

	var dstDigest []byte

	defer input.Close()

	dest := cmd.Dest

	link := false

	destStat, err := os.Lstat(dest)
	if err == nil {
		if destStat.IsDir() {
			dest = filepath.Join(dest, filepath.Base(src))
		} else {
			dstDigest, _ = md5file(dest)
		}

		link = destStat.Mode()&os.ModeSymlink != 0
	}

	rd := ResultData{
		"md5sum": Any(hex.EncodeToString(srcDigest)),
		"src":    Any(src),
		"dest":   Any(dest),
	}

	if dstDigest != nil && bytes.Equal(srcDigest, dstDigest) {
		changed := false

		if destStat.Mode() != srcStat.Mode() {
			changed = true
			if err := os.Chmod(dest, srcStat.Mode()); err != nil {
				return nil, err
			}
		}

		if ostat, ok := srcStat.Sys().(*syscall.Stat_t); ok {
			if estat, ok := destStat.Sys().(*syscall.Stat_t); ok {
				if ostat.Uid != estat.Uid || ostat.Gid != estat.Gid {
					changed = true
					os.Chown(dest, int(ostat.Uid), int(ostat.Gid))
				}
			}
		}

		return WrapResult(changed, rd), nil
	}

	tmp := fmt.Sprintf("%s.tmp.%d", cmd.Dest, os.Getpid())

	output, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return nil, err
	}

	defer output.Close()

	if _, err = io.Copy(output, input); err != nil {
		os.Remove(tmp)
		return nil, err
	}

	if link {
		os.Remove(dest)
	}

	if err := os.Chmod(tmp, srcStat.Mode()); err != nil {
		os.Remove(tmp)
		return nil, err
	}

	if ostat, ok := srcStat.Sys().(*syscall.Stat_t); ok {
		os.Chown(tmp, int(ostat.Uid), int(ostat.Gid))
	}

	err = os.Rename(tmp, dest)
	if err != nil {
		os.Remove(tmp)
		return nil, err
	}

	return WrapResult(true, rd), nil
}

type ScriptCmd struct {
	Script     string `tachyon:"command,required"`
	Creates    string `tachyon:"creates"`
	IgnoreFail bool   `tachyon:"ignore_failure"`
}

func (cmd *ScriptCmd) ParseArgs(s Scope, args string) (Vars, error) {
	if args == "" {
		return Vars{}, nil
	}

	return Vars{"command": Any(args)}, nil
}

func (cmd *ScriptCmd) Run(env *CommandEnv) (*Result, error) {
	if cmd.Creates != "" {
		if _, err := os.Stat(cmd.Creates); err == nil {
			r := NewResult(false)
			r.Add("rc", 0)
			r.Add("exists", cmd.Creates)

			return r, nil
		}
	}

	script := cmd.Script

	parts, err := shlex.Split(cmd.Script)
	if err == nil {
		script = parts[0]
	}

	path := env.Paths.File(script)

	_, err = os.Stat(path)
	if err != nil {
		return nil, err
	}

	runArgs := append([]string{"sh", path}, parts[1:]...)

	return runCmd(env, cmd.IgnoreFail, runArgs...)
}

func init() {
	RegisterCommand("command", &CommandCmd{})
	RegisterCommand("shell", &ShellCmd{})
	RegisterCommand("copy", &CopyCmd{})
	RegisterCommand("script", &ScriptCmd{})
}
