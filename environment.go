package tachyon

import (
	"errors"
	"io/ioutil"
	"os"
)

type Environment struct {
	Vars   Scope
	report Reporter
	config *Config
	tmpDir string

	Paths Paths
}

func NewEnv(s Scope, cfg *Config) *Environment {
	e := new(Environment)
	e.report = sCLIReporter
	e.Vars = s
	e.config = cfg

	d, err := ioutil.TempDir("", "tachyon")
	if err == nil {
		e.tmpDir = d
	}

	e.Paths = SimplePath{"."}

	return e
}

func (e *Environment) ReportJSON() {
	e.report = sJsonChunkReporter
}

var eNoTmpDir = errors.New("No tempdir available")

func (e *Environment) TempFile(prefix string) (*os.File, error) {
	if e.tmpDir == "" {
		return nil, eNoTmpDir
	}

	dest, err := ioutil.TempFile(e.tmpDir, prefix)
	return dest, err
}

func (e *Environment) Cleanup() {
	os.RemoveAll(e.tmpDir)
}

func (e *Environment) SetPaths(n Paths) Paths {
	cur := e.Paths
	e.Paths = n
	return cur
}
