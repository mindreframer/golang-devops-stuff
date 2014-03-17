package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Tag struct {
	Name, Value string
}

type Instances map[string][]Tag
type CloudInstances map[string]Instances

func getInstances(config Config) (clouds CloudInstances) {
	clouds = make(CloudInstances)

	var wg sync.WaitGroup
	var mux sync.RWMutex

	for name, cfg := range config {
		for k, v := range cfg {
			cfg["name"] = name
			cfg["provider"] = k

			if k == "provider" {
				switch v {
				case "aws":
					wg.Add(1)
					go func(name string, cfg StrMap) {
						mux.Lock()
						clouds[name] = getEC2Instances(cfg)
						mux.Unlock()
						wg.Done()
					}(name, cfg)
				case "digital_ocean":
					wg.Add(1)
					go func(name string, cfg StrMap) {
						mux.Lock()
						clouds[name] = getDigitalOceanInstances(cfg)
						mux.Unlock()
						wg.Done()
					}(name, cfg)
				default:
					log.Println("Unknown provider: ", v)
				}
			}
		}
	}

	wg.Wait()

	return
}

func getMatchedInstances(clouds CloudInstances, filter string) (matched []StrMap) {

	// Fuzzy matching, like SublimeText
	filter = strings.Join(strings.Split(filter, ""), ".*?")

	rHost := regexp.MustCompile(filter)

	for cloud, instances := range clouds {
		for addr, tags := range instances {
			for _, tag := range tags {
				if rHost.MatchString(cloud + tag.Value) {
					matched = append(matched, StrMap{
						"cloud":     cloud,
						"addr":      addr,
						"tag_name":  tag.Name,
						"tag_value": tag.Value,
					})

					break
				}
			}
		}
	}

	return
}

func formatMatchedInstance(inst StrMap) string {
	return "Cloud: " + inst["cloud"] + "\tMatched by: " + inst["tag_name"] + "=" + inst["tag_value"] + "\tAddr: " + inst["addr"]
}

func main() {
	config := readConfig()
	instances := getInstances(config)

	args := os.Args[1:len(os.Args)]

	user, hostname, arg_idx := getTargetHostname(args)

	match := getMatchedInstances(instances, hostname)

	if len(match) == 0 {
		fmt.Println("Can't find cloud instance, trying to connect anyway")
	} else if len(match) == 1 {
		hostname = match[0]["addr"]
		fmt.Println("Found clound instance:")
		fmt.Println(formatMatchedInstance(match[0]))
	} else {
		fmt.Println("Found multiple instances:")
		for i, host := range match {
			fmt.Println(strconv.Itoa(i+1)+") ", formatMatchedInstance(host))
		}
		fmt.Print("Choose instance: ")

		var i int
		_, err := fmt.Scanf("%d", &i)

		if err != nil || i > len(match)+1 {
			log.Fatal("Wrong index")
		}

		hostname = match[i-1]["addr"]
	}

	args[arg_idx] = joinHostname(user, hostname)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}
