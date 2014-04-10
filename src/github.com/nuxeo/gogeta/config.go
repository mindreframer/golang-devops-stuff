package main

import (
	"flag"
	"log"
)

type Config struct {
	port         int
	domainPrefix string
	envPrefix    string
	etcdAddress  string
	resolverType string
}

func parseConfig() *Config {
	config := &Config{}
	flag.IntVar(&config.port, "port", 7777, "Port to listen")
	flag.StringVar(&config.domainPrefix, "domainDir", "/nuxeo.io/domains", "etcd prefix to get domains")
	flag.StringVar(&config.envPrefix, "envDir", "/nuxeo.io/envs", "etcd prefix to get environments")
	flag.StringVar(&config.etcdAddress, "etcdAddress", "http://127.0.0.1:4001/", "etcd client host")
	flag.StringVar(&config.resolverType, "resolverType", "IoEtcd", "type of resolver (IoEtcd|Env|Dummy)")
	flag.Parse()

	log.Printf("Dumping Configuration")
	log.Printf("  listening port : %d", config.port)
	log.Printf("  domainPrefix : %s", config.domainPrefix)
	log.Printf("  envsPrefix : %s", config.envPrefix)
	log.Printf("  etcdAddress : %s", config.etcdAddress)
	log.Printf("  resolverType : %s", config.resolverType)

	return config
}
