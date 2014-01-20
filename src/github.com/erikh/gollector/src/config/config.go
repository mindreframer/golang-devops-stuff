package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"types"
)

func loadFromDir(configFile string) (cc types.CirconusConfig, err error) {
	plugins := make(types.PluginConfig)
	var found_main bool

	dir, err := os.Open(configFile)

	if err != nil {
		return cc, err
	}

	dirstat, err := dir.Stat()

	if err != nil {
		return cc, err
	}

	if !dirstat.IsDir() {
		panic(configFile + " is not a directory")
	}

	fi, err := dir.Readdir(0)

	if err != nil {
		return cc, err
	}

	for _, entry := range fi {
		if !entry.IsDir() {
			file, err := os.Open(filepath.Join(configFile, entry.Name()))

			if err != nil {
				return cc, err
			}

			content, err := ioutil.ReadAll(file)

			if err != nil {
				return cc, err
			}

			filename := entry.Name()

			if filename == "main.json" {
				err = json.Unmarshal(content, &cc)

				if err != nil {
					return cc, err
				}

				found_main = true
			} else {
				parts := strings.Split(filename, ".")
				un := types.ConfigMap{}
				err = json.Unmarshal(content, &un)

				if err != nil {
					return cc, err
				}

				plugins[parts[0]] = un
			}
		}
	}

	if found_main {
		cc.Plugins = plugins
	} else {
		panic("couldn't find main.json!")
	}

	return cc, nil
}

func loadFromFile(configFile string) (cc types.CirconusConfig, err error) {
	content, err := ioutil.ReadFile(configFile)

	if err != nil {
		return cc, err
	}

	err = json.Unmarshal(content, &cc)

	if cc.PollInterval == 0 {
		cc.PollInterval = 1
	}

	if cc.Facility == "" {
		cc.Facility = "daemon"
	}

	if cc.LogLevel == "" {
		cc.LogLevel = "info"
	}

	for k := range cc.Plugins {
		if len(cc.Plugins[k].Type) == 0 {
			old := cc.Plugins[k]
			cc.Plugins[k] = types.ConfigMap{
				Type:   k,
				Params: old.Params,
			}
		}
	}

	return cc, err
}

func Load(config string) (types.CirconusConfig, error) {
	stat, err := os.Stat(config)

	if err != nil {
		return types.CirconusConfig{}, err
	}

	if stat.IsDir() {
		return loadFromDir(config)
	} else {
		return loadFromFile(config)
	}
}

func Generate() {
	config := types.CirconusConfig{
		Listen:       ":8000",
		Username:     "gollector",
		Password:     "gollector",
		Facility:     "daemon",
		LogLevel:     "info",
		PollInterval: 5,
		Plugins:      make(types.PluginConfig),
	}

	for key, value := range types.Detectors {

		retval := value()

		if retval == nil {
			continue
		}

		if len(retval) == 0 {
			config.Plugins[key] = types.ConfigMap{
				Type:   key,
				Params: nil,
			}
			continue
		}

		for _, detected := range retval {
			newkey := strings.Join([]string{detected, key}, " ")
			config.Plugins[newkey] = types.ConfigMap{
				Type:   key,
				Params: detected,
			}
		}
	}

	res, err := json.MarshalIndent(config, "", "  ")

	if err != nil {
		fmt.Println("Error encountered while generating config:", err)
		os.Exit(1)
	}

	fmt.Println(string(res))
}
