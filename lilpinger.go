package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"lilpinger/config"
	"lilpinger/tools"
)

var (
	c = make(chan string)
)

func main() {
	//	read our text file of urls
	fileData, err := ioutil.ReadFile(config.Params.URLsFile)
	if err != nil {
		log.Fatal(err)
	}

	urls := []string{}
	//	split at new lines
	urls = strings.Split(string(fileData), "\n")

	for _, v := range urls {
		log.Println(v)
		if v != "" {
			go ping(v)
		}
	}

	//	output logs to the terminal
	for i := range c {
		fmt.Println(i)
	}
}

func ping(url string) {
	//	loop forever
	for {
		//	lag timer start
		start := time.Now()

		//	make our request
		res, err := http.Get(url)

		if err != nil {
			msg := "Error:" + err.Error()

			fmt.Println(msg)

			c <- msg
			reportError(msg)
		} else {
			lag := time.Since(start)
			var msg string

			//	running slow
			if lag > time.Duration(config.Params.LagThreshold)*time.Second {
				msg = url + " lag: " + lag.String()
				reportError(msg)
			}

			msg = url + ", lag: " + lag.String()
			c <- msg
		}

		res.Body.Close()
		time.Sleep(time.Duration(config.Params.PingInterval) * time.Second)
	}
}

func reportError(msg string) {
	tools.SendSMS(msg)
	tools.SendMail(msg)
}
