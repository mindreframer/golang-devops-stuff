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
	"os"
	"path/filepath"
	"strings"
	"time"
)

const buffered = 8

type DirWatcher interface {
	Updates() <-chan TaskFile
}

type TaskFile interface {
	// Name returns the task's base name, without any directory
	// prefix or .json suffix.
	Name() string

	// ConfigFileName returns the filename of the JSON file to read.
	// This returns the empty string when a file has been deleted.
	// TODO: make this more abstract, a ReadSeekCloser instead?
	ConfigFileName() string
}

var osDirWatcher DirWatcher // if nil, default polling impl is used

func dirWatcher() DirWatcher {
	if dw := osDirWatcher; dw != nil {
		return dw
	}
	return &pollingDirWatcher{dir: *configDir}
}

// pollingDirWatcher is the portable implementation of DirWatcher that
// simply polls the directory every few seconds.
type pollingDirWatcher struct {
	dir string
	c   chan TaskFile
}

func (w *pollingDirWatcher) Updates() <-chan TaskFile {
	if w.c == nil {
		w.c = make(chan TaskFile, buffered)
		go w.poll()
	}
	return w.c
}

func (w *pollingDirWatcher) poll() {
	last := map[string]time.Time{} // last modtime
	for {
		d, err := os.Open(w.dir)
		if err != nil {
			logger.Printf("Error opening directory %q: %v", w.dir, err)
			time.Sleep(15 * time.Second)
			continue
		}
		fis, err := d.Readdir(-1)
		d.Close()
		if err != nil {
			logger.Printf("Error reading directory %q: %v", w.dir, err)
			time.Sleep(15 * time.Second)
			continue
		}
		deleted := map[string]bool{}
		for n, _ := range last {
			deleted[n] = true // assume for now
		}
		for _, fi := range fis {
			name := fi.Name()
			if !strings.HasSuffix(name, ".json") {
				continue
			}
			if strings.HasPrefix(name, ".#") {
				continue
			}
			baseName := name[:len(name)-len(".json")]

			m := fi.ModTime()
			delete(deleted, baseName)
			if em, ok := last[baseName]; ok && em.Equal(m) {
				continue
			}
			logger.Printf("Updated config file: name = %q, modtime = %v", name, m)
			last[baseName] = m
			w.c <- diskFile{
				baseName: baseName,
				fileName: filepath.Join(w.dir, name),
				fi:       fi,
			}
		}
		for bn, _ := range deleted {
			w.c <- diskFile{baseName: bn}
		}
		time.Sleep(5 * time.Second)
	}
}

type diskFile struct {
	baseName string // base name, without .json suffix
	fileName string // relative path to *.json file
	fi       os.FileInfo
}

func (f diskFile) Name() string           { return f.baseName }
func (f diskFile) ConfigFileName() string { return f.fileName }
