[![Build Status](https://travis-ci.org/cloudfoundry/gorouter.png)](https://travis-ci.org/cloudfoundry/gorouter)

# gorouter

This repository contains the source code for a Go implementation of the Cloud
Foundry router.

This router is now used on CloudFoundry.com, replacing the old implementation.

## Summary

The original router can be found at cloudfoundry/router. The original router is
backed by nginx, that uses Lua code to connect to a Ruby server that -- based
on the headers of a client's request -- will tell nginx which backend it should
use. The main limitations in this architecture are that nginx does not support
non-HTTP (e.g. traffic to services) and non-request/response type traffic (e.g.
to support WebSockets), and that it requires a round trip to a Ruby server for
every request.

The Go implementation of the Cloud Foundry router is an attempt in solving
these limitations. First, with full control over every connection to the
router, it can more easily support WebSockets, and other types of traffic (e.g.
via HTTP CONNECT). Second, all logic is contained in a single process,
removing unnecessary latency.

## Getting started

The following instructions may help you get started with gorouter in a
standalone environment.

### External Dependencies

- Go should be installed and in the PATH
- GOPATH should be set as described in http://golang.org/doc/code.html
- [gnatsd](https://github.com/apcera/gnatsd) installed and in the PATH

### Development Setup

Download gorouter:
```bash
go get -v github.com/cloudfoundry/gorouter
cd $GOPATH/src/github.com/cloudfoundry/gorouter
```

To install exactly the dependecies vendored with gorouter, use [godep](https://github.com/tools/godep):

```bash
go get -v github.com/tools/godep
godep restore ./...
```

### Running Tests

We are using [Ginkgo](https://github.com/onsi/ginkgo), to run tests.

Running `scripts/test` will:
- Check for Go
- Check that GOPATH is set
- Download & Install gnatsd (or use the one already downloaded into the GOPATH)
- Update the PATH to prepend the godep workspace
- Install ginkgo (from the godep vendored sources into the godep workspace bin)
- Run all the tests with ginkgo (in random order, without benchmarks, using the vendored godep dependencies)

Any flags passed into `scripts/test` will be passed into ginkgo.

```bash
# run all the tests
scripts/test

# run only tests whose names match Registry
scripts/test -focus=Registry

# run only the tests in the registry package
scripts/test registry
```

To run the tests using GOPATH dependency sources (bypassing vendored dependencies):

```bash
ginkgo -r
```

### Building
Building creates an executable in the gorouter/ dir:

```bash
go build
```

### Installing
Installing creates an executable in the $GOPATH/bin dir:

```bash
go install
```

### Start

```bash
# Start NATS server in daemon mode
go get github.com/apcera/gnatsd
gnatsd &

# Start gorouter
gorouter
```

### Usage

When gorouter starts, it sends `router.start`. This message contains an
interval that other components should then send `router.register` on. If they
do not send a `router.register` for an amount of time considered "stale" by the
router, the routes are pruned. The default "staleness" is 2 minutes.

The format of this message is as follows:

```json
{
  "id": "some-router-id",
  "hosts": ["1.2.3.4"],
  "minimumRegisterIntervalInSeconds": 5
}
```

If a component comes online after the router, it must make a NATS request
called `router.greet` in order to determine the interval. The response to this
message will be the same format as `router.start`.

The format of route updates are as follows:

```json
{
  "host": "127.0.0.1",
  "port": 4567,
  "uris": [
    "my_first_url.vcap.me",
    "my_second_url.vcap.me"
  ],
  "tags": {
    "another_key": "another_value",
    "some_key": "some_value"
  }
}
```

Such a message can be sent to both the `router.register` subject to register
URIs, and to the `router.unregister` subject to unregister URIs, respectively.

```
$ nohup ruby -rsinatra -e 'get("/") { "Hello!" }' &
$ nats-pub 'router.register' '{"host":"127.0.0.1","port":4567,"uris":["my_first_url.vcap.me","my_second_url.vcap.me"],"tags":{"another_key":"another_value","some_key":"some_value"}}'
Published [router.register] : '{"host":"127.0.0.1","port":4567,"uris":["my_first_url.vcap.me","my_second_url.vcap.me"],"tags":{"another_key":"another_value","some_key":"some_value"}}'
$ curl my_first_url.vcap.me:8080
Hello!
```

### Instrumentation

Gorouter provides `/varz` and `/healthz` http endpoints for monitoring.

The `/routes` endpoint returns the entire routing table as JSON. Each route has an associated array of host:port entries.

Aside from the two monitoring http endpoints (which are only reachable via the status port), specifying the `User-Agent` header with a value of `HTTP-Monitor/1.1` also returns the current health of the router. This is particularly useful when performing healthchecks from a Load Balancer.

Because of the nature of the data present in `/varz` and `/routes`, they require http basic authentication credentials which can be acquired through NATS. The `port`, `user` and password (`pass` is the config attribute) can be explicitly set in the gorouter.yml config file's `status` section.

```
status:
  port: 8080
  user: some_user
  pass: some_password
```

Example interaction with curl:

```
curl -vvv -A "HTTP-Monitor/1.1" http://127.0.0.1/
* About to connect() to 127.0.0.1 port 80 (#0)
*   Trying 127.0.0.1... connected
> GET / HTTP/1.1
> User-Agent: HTTP-Monitor/1.1
> Host: 127.0.0.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Mon, 10 Feb 2014 00:55:25 GMT
< Transfer-Encoding: chunked
<
ok
* Connection #0 to host 127.0.0.1 left intact
* Closing connection #0

curl -vvv "http://someuser:somepass@127.0.0.1:8080/routes"
* About to connect() to 127.0.0.1 port 8080 (#0)
*   Trying 127.0.0.1...
* connected
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
* Server auth using Basic with user 'someuser'
> GET /routes HTTP/1.1
> Authorization: Basic c29tZXVzZXI6c29tZXBhc3M=
> User-Agent: curl/7.24.0 (x86_64-apple-darwin12.0) libcurl/7.24.0 OpenSSL/0.9.8r zlib/1.2.5
> Host: 127.0.0.1:8080
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: application/json
< Date: Mon, 25 Mar 2013 20:31:27 GMT
< Transfer-Encoding: chunked
<
{"0295dd314aaf582f201e655cbd74ade5.cloudfoundry.me":["127.0.0.1:34567"],"03e316d6aa375d1dc1153700da5f1798.cloudfoundry.me":["127.0.0.1:34568"]}
```

## Logs

The router's logging is specified in its YAML configuration file, in a [steno configuration format](http://github.com/cloudfoundry/steno#from-yaml-file).
The meanings of the router's log levels are as follows:

* `fatal` - An error has occurred that makes the current request unservicable.
Examples: the router can't bind to its TCP port, a CF component has published invalid data to the router.
* `warn` - An unexpected state has occurred. Examples: the router tried to publish data that could not be encoded as JSON
* `info`, `debug` - An expected event has occurred. Examples: a new CF component was registered with the router, the router has begun
to prune routes for stale droplets.

## Contributing

Please read the [contributors' guide](https://github.com/cloudfoundry/gorouter/blob/master/CONTRIBUTING.md)
