// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hook

import (
	"github.com/tsuru/config"
	"github.com/tsuru/gandalf/fs"
	"io"
	"os"
	"strings"
)

func createHookFile(path string, body io.Reader) error {
	file, err := fs.Filesystem().OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, body)
	if err != nil {
		return err
	}
	return nil
}

// Adds a hook script.
func Add(name string, repos []string, body io.Reader) error {
	config_param := "git:bare:template"
	if len(repos) > 0 {
		config_param = "git:bare:location"
	}
	path, err := config.GetString(config_param)
	if err != nil {
		return err
	}
	s := []string{path, "hooks", name}
	scriptPath := strings.Join(s, "/")
	if len(repos) > 0 {
		for _, repo := range repos {
			repo += ".git"
			s = []string{path, repo, "hooks", name}
			scriptPath = strings.Join(s, "/")
			err := createHookFile(scriptPath, body)
			if err != nil {
				return err
			}
		}
	} else {
		return createHookFile(scriptPath, body)
	}
	return nil
}
