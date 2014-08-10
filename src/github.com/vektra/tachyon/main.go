package tachyon

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
)

type Options struct {
	Vars        map[string]string `short:"s" long:"set" description:"Set a variable"`
	ShowOutput  bool              `short:"o" long:"output" description:"Show command output"`
	Host        string            `short:"t" long:"host" description:"Run the playbook on another host"`
	Development bool              `long:"dev" description:"Use a dev version of tachyon"`
	CleanHost   bool              `long:"clean-host" description:"Clean the host cache before using"`
	Debug       bool              `short:"d" long:"debug" description:"Show all information about commands"`
	Release     string            `long:"release" description:"The release to use when remotely invoking tachyon"`
	JSON        bool              `long:"json" description:"Output the run details in chunked json"`
	Install     bool              `long:"install" description:"Install tachyon a remote machine"`
}

var Release string = "dev"
var Arg0 string

func Main(args []string) int {
	var opts Options

	abs, err := filepath.Abs(args[0])
	if err != nil {
		panic(err)
	}

	Arg0 = abs

	parser := flags.NewParser(&opts, flags.Default)

	for _, o := range parser.Command.Group.Groups()[0].Options() {
		if o.LongName == "release" {
			o.Default = []string{Release}
		}
	}
	args, err = parser.ParseArgs(args)

	if err != nil {
		if serr, ok := err.(*flags.Error); ok {
			if serr.Type == flags.ErrHelp {
				return 2
			}
		}

		fmt.Printf("Error parsing options: %s", err)
		return 1
	}

	if !opts.Install && len(args) != 2 {
		fmt.Printf("Usage: tachyon [options] <playbook>\n")
		return 1
	}

	if opts.Host != "" {
		return runOnHost(&opts, args)
	}

	cfg := &Config{ShowCommandOutput: opts.ShowOutput}

	ns := NewNestedScope(nil)

	for k, v := range opts.Vars {
		ns.Set(k, v)
	}

	env := NewEnv(ns, cfg)
	defer env.Cleanup()

	if opts.JSON {
		env.ReportJSON()
	}

	playbook, err := NewPlaybook(env, args[1])
	if err != nil {
		fmt.Printf("Error loading plays: %s\n", err)
		return 1
	}

	cur, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to figure out the current directory: %s\n", err)
		return 1
	}

	defer os.Chdir(cur)
	os.Chdir(playbook.baseDir)

	runner := NewRunner(env, playbook.Plays)
	err = runner.Run(env)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running playbook: %s\n", err)
		return 1
	}

	return 0
}

func runOnHost(opts *Options, args []string) int {
	if opts.Install {
		fmt.Printf("=== Installing tachyon on %s\n", opts.Host)
	} else {
		fmt.Printf("=== Executing playbook on %s\n", opts.Host)
	}

	var playbook string

	if !opts.Install {
		playbook = args[1]
	}

	t := &Tachyon{
		Target:      opts.Host,
		Debug:       opts.Debug,
		Clean:       opts.CleanHost,
		Dev:         opts.Development,
		Playbook:    playbook,
		Release:     opts.Release,
		InstallOnly: opts.Install,
	}

	_, err := RunAdhocCommand(t, "")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return 1
	}

	return 0
}
