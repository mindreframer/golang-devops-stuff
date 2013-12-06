// +build ignore

package main

import (
	yagnats "github.com/cloudfoundry/yagnats"
	"log"
	"os"
	"os/signal"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	log.Printf("Sending messages...\n")

	client := yagnats.NewClient()
	err := client.Connect(&yagnats.ConnectionInfo{"127.0.0.1:4222", "nats", "nats"})
	if err != nil {
		log.Fatalf("Error connecting: %s\n", err)
	}

	bigbyte := make([]byte, 512000)
	go func() {
		for {
			client.Publish("foo", bigbyte)
		}
	}()

	<-c
	log.Printf("Bye!\n")
}
