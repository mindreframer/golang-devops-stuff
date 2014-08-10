package main

import (
	"errors"
	"flag"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
)

type Config struct {
	port          int
	domainPrefix  string
	servicePrefix string
	etcdAddress   string
	resolverType  string
	templateDir   string
	lastAccessInterval int
	client        *etcd.Client
	forceFwSsl	  bool
	UrlHeaderParam string
}

func (c *Config) getEtcdClient() (*etcd.Client, error) {
	if c.client == nil {
		c.client = etcd.NewClient([]string{c.etcdAddress})
		if !c.client.SyncCluster() {
			return nil, errors.New("Unable to sync with etcd cluster, check your configuration or etcd status")
		}
	}
	return c.client, nil
}

func parseConfig() *Config {
	config := &Config{}
	flag.IntVar(&config.port, "port", 7777, "Port to listen")
	flag.StringVar(&config.domainPrefix, "domainDir", "/domains", "etcd prefix to get domains")
	flag.StringVar(&config.servicePrefix, "serviceDir", "/services", "etcd prefix to get services")
	flag.StringVar(&config.etcdAddress, "etcdAddress", "http://127.0.0.1:4001/", "etcd client host")
	flag.StringVar(&config.resolverType, "resolverType", "IoEtcd", "type of resolver (IoEtcd|Env|Dummy)")
	flag.StringVar(&config.templateDir, "templateDir", "./templates", "Template directory")
	flag.StringVar(&config.UrlHeaderParam, "UrlHeaderParam", "", "Name of the param to inject the originating url")
	flag.IntVar(&config.lastAccessInterval,"lastAccessInterval",10,"Interval (in seconds to refresh last access time of a service")
	flag.BoolVar(&config.forceFwSsl, "forceFwSsl", false, "If not x-forwarded-proto set to https, then redirecto to the equivalent https url")
	flag.Parse()

	glog.Infof("Dumping Configuration")
	glog.Infof("  listening port : %d", config.port)
	glog.Infof("  domainPrefix : %s", config.domainPrefix)
	glog.Infof("  servicesPrefix : %s", config.servicePrefix)
	glog.Infof("  etcdAddress : %s", config.etcdAddress)
	glog.Infof("  resolverType : %s", config.resolverType)
	glog.Infof("  templateDir: %s", config.templateDir)
	glog.Infof("  lastAccessInterval: %d", config.lastAccessInterval)
	glog.Infof("  forceFwSsl: %t", config.forceFwSsl)
	glog.Infof("  UrlHeaderParam: %s", config.UrlHeaderParam)

	return config
}
