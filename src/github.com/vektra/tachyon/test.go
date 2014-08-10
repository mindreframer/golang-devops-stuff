package tachyon

import (
	"bytes"
	"fmt"
	"os"
)

func RunCapture(path string) (*Runner, string, error) {
	cfg := &Config{ShowCommandOutput: false}

	ns := NewNestedScope(nil)

	env := NewEnv(ns, cfg)
	defer env.Cleanup()

	playbook, err := NewPlaybook(env, path)
	if err != nil {
		fmt.Printf("Error loading plays: %s\n", err)
		return nil, "", err
	}

	var buf bytes.Buffer

	reporter := CLIReporter{out: &buf}

	runner := NewRunner(env, playbook.Plays)
	runner.SetReport(&reporter)

	cur, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	defer os.Chdir(cur)
	os.Chdir(playbook.baseDir)

	err = runner.Run(env)

	if err != nil {
		return nil, "", err
	}

	return runner, buf.String(), nil
}
