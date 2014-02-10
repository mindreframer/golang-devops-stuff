package main

import (
	"github.com/jingweno/gotask/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
