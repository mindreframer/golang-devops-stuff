package main

import (
	"fmt"
	"net"
	"strings"
)

// Replaces any occurrences of '127.0.0.1' or 'localhost' with the actual system IP-address.
func replaceLocalhostWithSystemIp(addresses *[]string) []string {
	out := []string{}
	for _, address := range *addresses {
		if address == "127.0.0.1" || strings.ToLower(address) == "localhost" {
			address = GetSystemIp()
		}
		out = append(out, address)
	}
	return out
}

func (this *Server) LoadBalancer_Add(conn net.Conn, addresses []string) error {
	addresses = replaceLocalhostWithSystemIp(&addresses)
	err := this.WithPersistentConfig(func(cfg *Config) error {
		cfg.LoadBalancers = this.UniqueStringsAppender(conn, cfg.LoadBalancers, addresses, "load-balancer", nil)
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{NewLogger(NewMessageLogger(conn), "[lb:add] ")}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}

func (this *Server) LoadBalancer_List(conn net.Conn) error {
	titleLogger, _ := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Listing load-balancers\n")

	return this.WithConfig(func(cfg *Config) error {
		for _, lb := range cfg.LoadBalancers {
			Logf(conn, "%v\n", lb)
		}
		return nil
	})
}

func (this *Server) LoadBalancer_Remove(conn net.Conn, addresses []string) error {
	addresses = replaceLocalhostWithSystemIp(&addresses)
	err := this.WithPersistentConfig(func(cfg *Config) error {
		cfg.LoadBalancers = this.UniqueStringsRemover(conn, cfg.LoadBalancers, addresses, "load-balancer", nil)
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{NewLogger(NewMessageLogger(conn), "[lb:remove] ")}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}
