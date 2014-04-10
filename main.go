package main

import "log"

const (
	progname = "gogeta"
)

func getResolver(c *Config) domainResolver {
	switch c.resolverType {
	case "Dummy":
		return &DummyResolver{}
	case "Env":
		return NewEnvResolver(c)
	default:
		return NewEtcdResolver(c)
	}
}

func main() {

	log.Printf("%s starting", progname)

	c := parseConfig()

	resolver := getResolver(c)
	resolver.init()

	p := NewProxy(c, resolver)
	p.start()

}
