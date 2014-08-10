// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	yagnats "github.com/cloudfoundry/yagnats"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	log.Printf("Receiving messages...\n")

	client := yagnats.NewClient()
	err := client.Connect(&yagnats.ConnectionInfo{
		Addr:     "127.0.0.1:4222",
		Username: "nats",
		Password: "nats",
	})
	if err != nil {
		log.Fatalf("Error connecting: %s\n", err)
	}

	seen := 0

	client.Subscribe("foo", func(msg *yagnats.Message) {
		for i := 0; i < 1000000; i++ {
			fmt.Printf("")
		}
		seen += 1
		fmt.Printf("got it! %d\n", seen)
	})

	<-c
	log.Printf("Messages processed: %d\n", seen)
}
