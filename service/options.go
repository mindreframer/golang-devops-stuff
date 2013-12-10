package service

import (
	"flag"
	"fmt"
	"github.com/mailgun/vulcan/backend"
	"regexp"
	"strconv"
	"time"
)

// Parses service options from the command line
func parseOptions() (*serviceOptions, error) {
	options := &serviceOptions{}

	flag.Var(&options.controlServers, "c", "HTTP control server url")
	flag.StringVar(&options.backend, "b", "memory", "Backend type e.g. 'cassandra' or 'memory'")
	flag.StringVar(&options.loadBalancer, "lb", "cassandra", "Loadbalancer algo, e.g. 'random'")

	flag.StringVar(&options.host, "h", "localhost", "Host to bind to")
	flag.IntVar(&options.httpPort, "p", 8080, "HTTP port to bind to")

	flag.StringVar(&options.pidPath, "pid", "", "pid file path")

	flag.Var(&options.cassandraServers, "csnode", "Cassandra nodes to connect to")
	flag.StringVar(&options.cassandraKeyspace, "cskeyspace", "", "Cassandra keyspace")

	flag.BoolVar(&options.cassandraCleanup, "cscleanup", false, "Whethere to perform periodic cassandra cleanups")
	flag.Var(&options.cassandraCleanupOptions, "cscleanuptime", "Cassandra cleanup utc time of day in form: HH:MM")

	flag.DurationVar(&options.cleanupPeriod, "logcleanup", time.Duration(24)*time.Hour, "How often should we remove unused golang logs (e.g. 24h, 1h, 7h)")

	flag.Parse()

	return options, nil
}

type serviceOptions struct {
	// Pid path
	pidPath string
	// Control servers to bind to
	controlServers listOptions
	backend        string
	loadBalancer   string

	// Host and port to bind to
	host     string
	httpPort int

	// Cassandra specific stuff
	cassandraServers        listOptions
	cassandraKeyspace       string
	cassandraCleanup        bool
	cassandraCleanupOptions cleanupOptions

	// How often should we clean up golang old logs
	cleanupPeriod time.Duration
}

// Helper to parse options that can occur several times, e.g. cassandra nodes
type listOptions []string

func (o *listOptions) String() string {
	return fmt.Sprint(*o)
}

func (o *listOptions) Set(value string) error {
	*o = append(*o, value)
	return nil
}

// Helper to parse cleanup time that is supplied as hh:mm format
// and represents utc time of the day when to launch cleanup procedures
type cleanupOptions struct {
	T *backend.CleanupTime
}

func (o *cleanupOptions) String() string {
	if o.T != nil {
		return fmt.Sprintf("%0d:%0d", o.T.Hour, o.T.Minute)
	}
	return "not set"
}

func (o *cleanupOptions) Set(value string) error {
	re := regexp.MustCompile(`(?P<hour>\d+):(?P<minute>\d+)`)
	values := re.FindStringSubmatch(value)
	if values == nil {
		return fmt.Errorf("Invalid format, expected HH:MM")
	}
	hour, err := strconv.Atoi(values[1])
	if err != nil {
		return err
	}
	minute, err := strconv.Atoi(values[2])
	if err != nil {
		return err
	}
	o.T = &backend.CleanupTime{Hour: hour, Minute: minute}
	return nil
}
