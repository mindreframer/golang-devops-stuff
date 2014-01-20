package main

import (
	"config"
	"fmt"
	"logger"
	"os"
	"web"
)

func checkUsage() {
	if len(os.Args) < 2 {
		fmt.Println("usage:", os.Args[0], "<config file or directory> or 'generate'")
		os.Exit(2)
	}
}

func main() {
	checkUsage()

	if os.Args[1] == "generate" {
		config.Generate()
		os.Exit(0)
	} else {
		this_config, err := config.Load(os.Args[1])

		if err != nil {
			fmt.Println("Error while loading config file:", err)
			os.Exit(1)
		}

		log := logger.Init(this_config.Facility, this_config.LogLevel)

		result := web.Start(this_config.Listen, this_config, log)

		log.Log("crit", fmt.Sprintf("Failed to start: %s", result))
		log.Close()
	}
}
