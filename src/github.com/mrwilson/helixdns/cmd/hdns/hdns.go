package main

import (
  "github.com/mrwilson/helixdns"
  "os"
  "os/signal"
  "syscall"
  "log"
  "flag"
)

var config struct {
  Port int
  EtcdAddress string
}

func init() {
  flag.IntVar(&config.Port, "port", 9000, "port to run server on")
  flag.StringVar(&config.EtcdAddress, "etcd-address", "http://localhost:4001/", "address of etcd instance")
}

func main() {
  flag.Parse()

  server := helixdns.Server(config.Port, config.EtcdAddress)

  go func() {
    server.Start()
  }()

  sig := make(chan os.Signal)
  signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
  for {
    select {
    case s := <-sig:
      log.Fatalf("Signal (%d) received, stopping\n", s)
    }
  }
}
