package main

import (
	"flag"
	"fmt"
	"github.com/stripe-ctf/octopus/director"
	"github.com/stripe-ctf/octopus/harness"
	"github.com/stripe-ctf/octopus/log"
	"github.com/stripe-ctf/octopus/state"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: %s [options] [...args]

Octopus will run your sqlcluster and issue queries to it. It simulates
a lossy network, and to make things even more fun, will turn loose the
following chaotic monkeys:

  NET MONKEYS
  - latency: adjusts base latency between nodes
  - jitter: adjusts latency variance between nodes
  - lagsplit: spikes latency along a network partition
  - link: destroys individual links between nodes
  - netsplit: destroys links along a network partition

  NODE MONKEYS
  - freeze: stops nodes (by sending SIGTSTP)
  - murder: ruthlessly murders nodes (by sending SIGTERM)

Any positional arguments given to Octopus will be passed through to
SQLCluster. This should be mostly useful for passing a -v flag, like
so:

  ./octopus -- -v

OPTIONS:
`, os.Args[0])
		flag.PrintDefaults()
	}
	state.AddFlags()
	flag.Parse()
	state.AfterParse()

	// Handle SIGINT
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	d := director.NewDirector()
	if state.Dryrun() {
		d.Dryrun()
		return
	}

	// Create the working directory
	if err := os.MkdirAll(state.Root(), os.ModeDir|0755); err != nil {
		log.Fatal(err)
	}

	d.Start()

	go func() {
		select {
		case <-sigchan:
			log.Println("Terminating due to signal")
		case <-state.WaitGroup().Quit:
			// Someone else requested an exit
		case <-time.After(state.Duration()):
			// Time's up!
			log.Printf("The allotted %s have elapsed. Exiting!", state.Duration())
		}
		state.WaitGroup().Exit()
	}()

	h := harness.New(d.Agents())
	h.Start()

	d.StartMonkeys()

	<-state.WaitGroup().Quit

	if state.Write() != "" {
		results := state.JSONResults()
		if err := ioutil.WriteFile(state.Write(), results, 0755); err != nil {
			log.Fatalf("Could not write resuts: %s", err)
		}
	} else {
		results := state.PrettyPrintResults()
		log.Println(results)
	}

	state.WaitGroup().Exit()
}
