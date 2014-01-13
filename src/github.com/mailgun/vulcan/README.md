[![Build Status](https://travis-ci.org/mailgun/vulcan.png)](https://travis-ci.org/mailgun/vulcan)
[![Build Status](https://drone.io/github.com/mailgun/vulcan/status.png)](https://drone.io/github.com/mailgun/vulcan/latest)
[![Coverage Status](https://coveralls.io/repos/mailgun/vulcan/badge.png?branch=master)](https://coveralls.io/r/mailgun/vulcan?branch=master)

Status
=======
Don't use it in production, early adopters and hackers are welcome


Proxy for HTTP services
-----------------------

Vulcan is a proxy built for APi's specific needs that are usually different from website's needs. It is a proxy that you program in JavaScript.

```javascript
function handle(request){
    return {upstreams: ["http://localhost:5000", "http://localhost:5001"]}
}
```

How slow can your proxy be?
---------------------------
One wants proxies to be fast, but in case of services proxy is rarely a bottleneck, whereas DB and filesystem are.
Vulcan supports rate limiting using memory, Cassandra or Redis backends, so your service can introduce proper account-specific rates and expectations right from the start.

```javascript
function handle(request){
    return {
        failover: true,
        upstreams: ["http://localhost:5000", "http://localhost:5001"],
        rates: {request.ip: ["10 requests/second", "1000 KB/second"]}
    }
}
```

Discover FTW!
-------------

Storing upstreams in files is ok up to a certain extent. On the other hand, keeping upstreams in a discovery service simplifies deployment and configuration management. Vulcan supports Etcd or Zookeeper:

```javascript
function handle(request){
    return {
        upstreams: discover("/upstreams"),
        rates: {request.ip: ["10 requests/second", "1000 KB/second"]}
    }
}
```

Caching and Auth
-----------------

Auth is hard and you don't want every endpoint to implement auth. It's better to implement auth endpoint once, and make proxy deal with it. As a bonus you can cache results using memory, Redis or Cassandra, reducing load on the databases holding account creds.

```javascript
function handle(request){
    response = get(discover("/auth-endpoints"), {auth: request.auth}, {cache: true})
    if(!response.code == 200) {
        return response
    }
    return {
        upstreams: discover("/upstreams"),
        rates: {request.ip: ["10 requests/second", "1000 KB/second"]}
    }
}
```

And many more advanced features you'd need when writing APIs, like Metrics and Failure detection. Read on!


__Development setup__

Mailing list: https://groups.google.com/forum/#!forum/vulcan-proxy

__Install go__

(http://golang.org/doc/install)

__Get vulcan and install deps__
 
```bash
# set your GOPATH to something reasonable.
export GOPATH=~/projects/vulcan
cd $GOPATH
go get github.com/mailgun/vulcan

make -C ./src/github.com/mailgun/vulcan deps
cd ./src/github.com/mailgun/vulcan
```

__Run in devmode__
 
```bash 
make run
```

__Cassandra__

Cassandra-based throttling is a generally good idea, as it provides reliable distributed
counters that can be shared between multiple instances of vulcan. Vulcan provides auto garbage collection
and cleanup of the counters.

Tested on versions >= 1.2.5

Usage
-------

```bash
vulcan \
       -h=0.0.0.0\                  # interface to bind to
       -p=4000\                     # port to listen on
       -c=http://localhost:5000 \   # control server url#1
       -c=http://localhost:5001 \   # control server url#2, for redundancy
       -stderrthreshold=INFO \      # log info, from glog
       -logtostderr=true \          # log to stderror
       -logcleanup=24h \            # clean up logs every 24 hours
       -log_dir=/var/log/           # keep log files in this folder
       -pid=/var/run/vulcan.pid     # create pid file
       -lb=roundrobin \             # use round robin load balancer
       -b=cassandra \               # use cassandra for throttling
       -cscleanup=true \            # cleanup old counters
       -cscleanuptime=19:05 \       # cleanup counters 19:05 UTC every day
       -csnode=localhost  \         # cassandra node, can be multiple
       -cskeyspace=vulcan_dev       # cassandra keyspace
```

Development
-----------
To run server in development mode:

```bash
make run
```

To run tests

```bash
make test
```

To run tests with coverage:

```bash
make coverage
```

To cleanup temp folders

```bash
make clean
```

Status
------
Initial development done, loadtesting at the moment and fixing quirks. 
