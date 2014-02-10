package cli

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/jingweno/gotask/build"
	"log"
	"os"
)

var (
	debugFlag = cli.BoolFlag{Name: "debug", Usage: "run in debug mode"}
)

func NewApp() *cli.App {
	cmds, err := parseCommands()
	if err != nil {
		log.Fatal(err)
	}

	app := cli.NewApp()
	app.Name = "gotask"
	app.Usage = "Build tool in Go"
	app.Version = Version
	app.Commands = cmds
	app.Flags = []cli.Flag{
		generateFlag,
		compileFlag,
		debugFlag,
	}
	app.Action = func(c *cli.Context) {
		if c.Bool("g") || c.Bool("generate") {
			fileName, err := generateNewTask()
			if err == nil {
				fmt.Printf("create %s\n", fileName)
			} else {
				log.Fatal(err)
			}

			return
		}

		if c.Bool("c") || c.Bool("compile") {
			err := compileTasks(c.Bool("debug"))
			if err != nil {
				log.Fatal(err)
			}

			return
		}

		args := c.Args()
		if len(args) > 0 {
			cli.ShowCommandHelp(c, args[0])
		} else {
			cli.ShowAppHelp(c)
		}
	}

	return app
}

func parseCommands() (cmds []cli.Command, err error) {
	source, err := os.Getwd()
	if err != nil {
		return
	}

	parser := build.NewParser()
	taskSet, err := parser.Parse(source)
	if err != nil {
		return
	}

	for _, t := range taskSet.Tasks {
		task := t
		cmd := cli.Command{
			Name:        task.Name,
			Usage:       task.Usage,
			Description: task.Description,
			Flags:       append(t.ToCLIFlags(), debugFlag),
			Action: func(c *cli.Context) {
				err := runTasks(os.Args[1:], c.Bool("debug"))
				if err != nil {
					log.Fatal(err)
				}
			},
		}

		cmds = append(cmds, cmd)
	}

	return
}

func runTasks(args []string, isDebug bool) (err error) {
	sourceDir, err := os.Getwd()
	if err != nil {
		return
	}

	filteredArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "--debug" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	err = build.Run(sourceDir, filteredArgs, isDebug)
	return
}
