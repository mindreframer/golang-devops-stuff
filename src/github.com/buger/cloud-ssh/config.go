package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

type Config map[string]StrMap
type StrMap map[string]string

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func readConfig() (config Config) {
	config = make(Config)

	prefferedPaths := []string{
		"./cloud-ssh.yaml",
		userHomeDir() + "/.ssh/cloud-ssh.yaml",
		"/etc/cloud-ssh.yaml",
	}

	var content []byte

	for _, path := range prefferedPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Println("Found config:", path)
			content, err = ioutil.ReadFile(path)

			if err != nil {
				log.Fatal("Error while reading config: ", err)
			}

			break
		}
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		config["default"] = make(StrMap)
		config["default"]["access_key"] = os.Getenv("AWS_ACCESS_KEY_ID")
		config["default"]["secret_key"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
		config["default"]["region"] = os.Getenv("AWS_REGION")
		config["default"]["provider"] = "aws"
	}

	if len(content) == 0 {
		if len(config) == 0 {
			fmt.Println("Can't find any configuration or ENV variables. Check http://github.com/buger/cloud-ssh for documentation.")
		}
		return
	} else if err := yaml.Unmarshal(content, &config); err != nil {
		log.Fatal(err)
	}

	return
}
