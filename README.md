# Gollector

Gollector is a metrics collector that emits JSON responses, which can be
consumed by monitoring systems such as [Circonus](http://circonus.com). The
flexibility of the JSON responses leads to many monitoring possibilities, such
as the included 'gstat', which is an n-host iostat-alike for all metrics
gollector is collecting.

Here's a graph generated in Circonus from data provided by Gollector:

![An Example](graph.png)

Most of the built-in collectors are linux-only for now, and probably the future
unless pull requests happen. Many plugins very likely require a 3.0 or later
kernel release due to dependence on system structs and other deep voodoo.

Gollector does not need to be run as root to collect its metrics. For things
that need root, or work with additional data sources (such as data stores),
check out the sister project [Gollector Monitors](https://github.com/erikh/gollector-monitors).

Gollector also now supports Graphite! `make gollector-graphite` or just `make`
will build the program, which bridges the two services. Please see the tool's
usage information (no arguments) for assistance using this feature.

Unlike other collectors that use fat tools like `netstat` and `df` which can
take expensive resources on loaded systems, Gollector opts to use the C
interfaces directly when it can. This allows it to keep a very small footprint;
with the go runtime, it clocks in just above 5M resident and unnoticeable CPU
usage at the time of writing. The agent can sustain over 8000qps with a
benchmarking tool like `wrk`, so it will be plenty fine getting hit once per
minute, or even once per second.

## Quick Start

In the gollector directory on a Linux machine with kernel 3.0 or better:

```bash
$ make
$ ./gollector generate > gollector.json
$ ./gollector gollector.json &
$ ./gstat -hosts localhost -metric "load_average"
```

Should yield an array of floats that contain your current load average.

```bash
$ curl http://gollector:gollector@localhost:8000/
```

Will yield a json object of all current metrics.


## Wiki

Our [wiki](https://github.com/erikh/gollector/wiki) contains tons of information
on advanced configuration, usage, and even tools you can use with Gollector.
Check it out!

## License

* MIT (C) 2013 Erik Hollensbe
