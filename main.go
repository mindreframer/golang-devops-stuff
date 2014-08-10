package main

import (
	"github.com/golang/glog"
)

const (
	progname = "gogeta"
)

func getResolver(c *Config) (domainResolver, error) {
	switch c.resolverType {
	case "Dummy":
		return &DummyResolver{}, nil
	case "Env":
		return NewEnvResolver(c), nil
	default:
		r, err := NewEtcdResolver(c)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}

func main() {

	glog.Infof("%s starting", progname)

	c := parseConfig()

	resolver, error := getResolver(c)
	if error != nil {
		panic(error)
	} else {

		resolver.init()

		p := NewProxy(c, resolver)
		p.start()
	}

}
