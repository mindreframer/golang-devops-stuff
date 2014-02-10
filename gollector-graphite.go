package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const PLAINTEXT_FORMAT = "servers.%s %f %d\n"

func formatKey(orig_key, key string) string {
	new_key := strings.Replace(orig_key+"."+key, " ", "_", -1)
	new_key = strings.Replace(new_key, "/", "_", -1)
	return regexp.MustCompile("[()]").ReplaceAllString(new_key, "")
}

func writeMetric(conn net.Conn, key string, value interface{}) {
	str := fmt.Sprintf(PLAINTEXT_FORMAT, key, value, time.Now().Unix())
	conn.Write([]byte(str))
}

func iterateNav(conn net.Conn, value_type reflect.Type, key string, value interface{}) {
	if value_type.Kind() == reflect.Bool {
		// XXX skip bools
	} else if value_type.Kind() == reflect.Map && value_type.String() == "map[string]interface {}" {
		navigateJSONMap(conn, key, value.(map[string]interface{}))
	} else if value_type.String() == "[]interface {}" {
		navigateJSONArray(conn, key, value.([]interface{}))
	} else {
		writeMetric(conn, key, value)
	}
}

func navigateJSONArray(conn net.Conn, key string, array []interface{}) {
	for i, value := range array {
		value_type := reflect.TypeOf(value)

		iterateNav(conn, value_type, key+".index_"+fmt.Sprintf("%d", i), value)
	}
}

func navigateJSONMap(conn net.Conn, orig_key string, json_rep map[string]interface{}) {
	for key, value := range json_rep {
		new_key := formatKey(orig_key, key)
		value_type := reflect.TypeOf(value)

		iterateNav(conn, value_type, new_key, value)
	}
}

func main() {
	connect := flag.String("connect", "localhost:2003", "Graphite plaintext protocol to emit to")
	interval := flag.Int("interval", 60, "Frequency of poll (in seconds")

	flag.Parse()

	if len(flag.Args()) < 1 {
		os.Stderr.WriteString("Must supply at least one gollector endpoint as a url")
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", *connect)

	if err != nil {
		panic(err)
	}

	for _, gollector_url := range flag.Args() {
		go func() {
			for {

				this_url, err := url.Parse(gollector_url)

				if err != nil {
					panic(err)
				}

				resp, err := http.Get(gollector_url)

				if err != nil {
					time.Sleep(1 * time.Second)
					continue
				}

				content, err := ioutil.ReadAll(resp.Body)
				resp.Body.Close()

				if err != nil {
					time.Sleep(1 * time.Second)
					continue
				}

				json_rep := map[string]interface{}{}
				err = json.Unmarshal(content, &json_rep)

				if err != nil {
					time.Sleep(1 * time.Second)
					continue
				}

				navigateJSONMap(conn, strings.Split(this_url.Host, ".")[0], json_rep)
				time.Sleep(time.Duration(*interval) * time.Second)
			}
		}()
	}

	for {
		time.Sleep(1 * time.Hour)
	}
}
