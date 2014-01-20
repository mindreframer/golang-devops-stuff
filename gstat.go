package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type options struct {
	hosts    string
	username string
	password string
	port     uint
	metric   string
}

func poll(host string, metric string, opts options, result_chan chan [3]string) {
	host_url := fmt.Sprintf("http://%s:%s@%s:%d", opts.username, opts.password, host, opts.port)
	payload, err := json.Marshal(map[string]string{"Name": metric})
	resp, err := http.Post(host_url, "application/json", strings.NewReader(string(payload)))

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	json, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	result_chan <- [3]string{host, metric, string(json)}
}

func main() {
	var opts options

	flag.StringVar(&opts.hosts, "hosts", "", "comma-separated host list")
	flag.StringVar(&opts.username, "username", "gollector", "username to use for authentication")
	flag.StringVar(&opts.password, "password", "gollector", "password to use for authentication")
	flag.UintVar(&opts.port, "port", 8000, "port to use for connection")
	flag.StringVar(&opts.metric, "metric", "", "comma-separated list of metrics to fetch for polling")
	flag.Parse()

	if opts.hosts == "" {
		fmt.Println("Please provide a list of hosts to monitor")
		os.Exit(1)
	}

	if opts.metric == "" {
		fmt.Println("Please provide a metric to monitor")
		os.Exit(1)
	}

	if opts.username == "" || opts.password == "" {
		fmt.Println("Please supply a username and password for authentication to the gollector agent(s)")
		os.Exit(1)
	}

	result_chan := make(chan [3]string)
	results := make(map[string]map[string]string)
	var host_keys []string
	var metric_keys []string
	metric_key_map := make(map[string]bool)
	hosts := strings.Split(opts.hosts, ",")
	metrics := strings.Split(opts.metric, ",")

	for {
		fmt.Println()

		for _, host := range hosts {
			for _, metric := range metrics {
				go poll(host, metric, opts, result_chan)
			}
		}

		for i := 0; i < len(hosts)*len(metrics); i++ {
			result := <-result_chan

			if results[result[0]] == nil {
				results[result[0]] = map[string]string{}
				host_keys = append(host_keys, result[0])
			}

			results[result[0]][result[1]] = result[2]
			metric_key_map[result[1]] = true
		}

		for m, _ := range metric_key_map {
			metric_keys = append(metric_keys, m)
		}

		sort.Strings(host_keys)
		sort.Strings(metric_keys)

		for _, m := range metric_keys {
			for _, h := range host_keys {
				fmt.Printf("%s %s: %s\n", h, m, results[h][m])
			}
		}

		time.Sleep(1 * time.Second)
		metric_keys = []string{}
	}
}
