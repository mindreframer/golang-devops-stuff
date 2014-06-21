/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// LaunchRequest is a subset of exec.Cmd plus the addition of Uid/Gid.
// This structure is gob'd and base64'd and sent to the child process
// in the environment variable _RUNSIT_LAUNCH_INFO.  The child then
// drops root and execs itself to be the requested process.
type LaunchRequest struct {
	Uid      int   // or 0 to not change
	Gid      int   // or 0 to not change
	Gids     []int // supplemental
	Path     string
	Env      []string
	Argv     []string // must include Path as argv[0]
	Dir      string
	NumFiles int // new nfile fd rlimit, or 0 to not change
}

func (lr *LaunchRequest) start(extraFiles []*os.File) (cmd *exec.Cmd, outPipe, errPipe io.ReadCloser, err error) {
	var buf bytes.Buffer
	b64enc := base64.NewEncoder(base64.StdEncoding, &buf)
	err = gob.NewEncoder(b64enc).Encode(lr)
	b64enc.Close()
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			for _, p := range []io.ReadCloser{outPipe, errPipe} {
				if p != nil {
					p.Close()
				}
			}
		}
	}()

	cmd = exec.Command(os.Args[0])
	cmd.Env = append(cmd.Env, "_RUNSIT_LAUNCH_INFO="+buf.String())
	cmd.ExtraFiles = extraFiles
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	outPipe, err = cmd.StdoutPipe()
	if err != nil {
		return
	}
	errPipe, err = cmd.StderrPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	return cmd, outPipe, errPipe, nil
}

func MaybeBecomeChildProcess() {
	lrs := os.Getenv("_RUNSIT_LAUNCH_INFO")
	if lrs == "" {
		return
	}
	defer os.Exit(2) // should never make it this far, though

	lr := new(LaunchRequest)
	d := gob.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(lrs)))
	err := d.Decode(lr)
	if err != nil {
		log.Fatalf("Failed to decode LaunchRequest in child: %v", err)
	}
	if lr.NumFiles != 0 {
		var lim syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
			log.Fatalf("failed to get NOFILE rlimit: %v", err)
		}
		noFile := rlim_t(lr.NumFiles)
		lim.Cur = noFile
		lim.Max = noFile
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
			log.Fatalf("failed to set NOFILE rlimit: %v", err)
		}
	}
	if lr.Gid != 0 {
		if err := syscall.Setgid(lr.Gid); err != nil {
			log.Fatalf("failed to Setgid(%d): %v", lr.Gid, err)
		}
	}
	if len(lr.Gids) != 0 {
		if err := syscall.Setgroups(lr.Gids); err != nil {
			log.Printf("setgroups: %v", err)
		}
	}
	if lr.Uid != 0 {
		if err := syscall.Setuid(lr.Uid); err != nil {
			log.Fatalf("failed to Setuid(%d): %v", lr.Uid, err)
		}
	}
	if lr.Path != "" {
		err = os.Chdir(lr.Dir)
		if err != nil {
			log.Fatalf("failed to chdir to %q: %v", lr.Dir, err)
		}
	}
	err = syscall.Exec(lr.Path, lr.Argv, lr.Env)
	log.Fatalf("failed to exec %q: %v", lr.Path, err)
}
