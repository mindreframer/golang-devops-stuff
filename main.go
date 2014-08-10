package main

import (
  "os"
  "os/signal"
  "syscall"
  "log"
  "flag"
)

var config struct {
  Port int
  EtcdAddress string
  ForwardingNameServer string
}

func init() {
  flag.IntVar(&config.Port, "port", 9000, "port to run server on")
  flag.StringVar(&config.EtcdAddress, "etcd-address", "http://localhost:4001/", "address of etcd instance")
  flag.StringVar(&config.ForwardingNameServer, "forward", "", "address of forwarding nameserver")
}

func main() {
  flag.Parse()

  var server HelixServer

  if config.ForwardingNameServer != "" {
    server = ForwardingServer(config.Port, config.EtcdAddress, config.ForwardingNameServer)
  } else {
    server = Server(config.Port, config.EtcdAddress)
  }

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
