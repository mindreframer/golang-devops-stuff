Warning!
--------
**Vulcan is under heavy development and API is likely to change!**
**Beware when integrating and using it at this stage!**

[![Build Status](https://travis-ci.org/mailgun/vulcan.png)](https://travis-ci.org/mailgun/vulcan)

Development coordination: https://trello.com/b/DLlP2CKX/vulcan

Mailing list: https://groups.google.com/forum/#!forum/vulcan-proxy

Vulcan
------

Programmatic HTTP reverse proxy for creating JSON-based API services with:

* Rate limiting
* Load balancing
* Early error detection, failover and alerting
* Metrics
* Dynamic service discovery

__Note__

Metrics and service discovery are not implemented yet, rate limiting and load balancing with failover are here.

Rationale
---------

There's a room for a proxy that would make lives of people writing modern API services a bit simpler.

Project status
--------------

* Active development (Configuration files and API may change, once we finalize the idea, the config file will be freezed)
* Vulcan is currently in production at Mailgun serving moderate loads on some services (< 1K requests per second)

Request flow
------------

* Client request arrives to the Vulcan.
* Vulcan extracts request information and asks control server what to do with the request.
* Vulcan denies or throttles and routes the request according to the instructions from the control server.
* If the upstream fails, Vulcan can optionally forward the request to the next upstream.


Authorization
-------------

Vulcan sends the following request info to the control server:

|Name    |Descripton                |
|--------|--------------------------|
|username| HTTP auth username       |
|password| HTTP auth password       |
|protocol| protocol (SMTP/HTTP)     |
|url     | original request url     |
|headers | (JSON encoded dictionary)|
|length  | request size in bytes    |


Control server can deny the request by responding with non 200 response code. 
In this case the exact control server response will be proxied to the client.
Otherwise, control server replies with JSON understood by the proxy. See Routing section for details.

Routing & Rate limiting
--------------------

If the request is good to go, control server replies with json in the following format:

```javascript
{
        "tokens": [
            {
                "id": "hello",
                "rates": [
                    {"increment": 1, "value": 10, "period": "minute"}
                ]
            }
       ],
       "upstreams": [
            {
                "url": "http://localhost:5000/upstream",
                'rates': [
                    {"increment": 1, "value": 2, "period": "minute"}
                 ]
            },
            {
                "url": "http://localhost:5000/upstream2",
                "rates": [
                    {"increment": 1, "value": 4, "period": "minute"}
                 ]
            }
       ])
}

```

* In this example all requests will be throttled by the same token 'hello', with maximum 10 hits per minute total.
* The request can be routed to one of the two upstreams, the first upstream allows max 2 requests per minute, the second one allows 4 requests per minute.

In case if all upstreams are busy or tokens rates are not allowing the request to proceed, Vulcan replies with json-encoded response:

```javascript
{
        "retry-seconds": 20,
        ...
}

```

Vulcan tells client when the next request can succeed, so clients can embrace this data and reschedule the request in 20 seconds. Note that this
is an estimate and does not guarantee that request will succeed, it guarantees that request would not succeed if executed before waiting given amount
of seconds. It allows not to waste resources and keep trying.

Failover
--------

* In case if control server fails, vulcan automatically queries the next available server.
* In case of upstream being slow or unresponsive, Vulcan can retry the request with the next upstream. 

This option turned on by the failover flag in the control response:


```javascript
{
        "failover": {
            "active": true, // Activate fallback for this request
            "codes": [410, 411] // Optional fallback codes
        },
        ...
}

```

* In case if upstream unexpectedly fails, Vulcan will retry the same request on the next upstream selected by the load balancer
* Notice "codes" parameter. Once vulcan sees these response codes from the list, it will replay the request instead of proxying it to the client.
This allows graceful service deployments.

__Note__

Failover allows fast deployments of the underlying applications, however it requires that the request would be idempotent, i.e. can be safely retried several times. Read more about the term here: http://stackoverflow.com/questions/1077412/what-is-an-idempotent-operation

E.g. most of the GET requests are idempotent as they don't change the app state, however you should be more careful with POST requests,
as they may do some damage if repeated.

Failovers can also lead to the cascading failures. Imagine some bad request killing your service, in this case failover will kill all upstreams! That's why make sure you return limited amount of upstreams with the control response in case of failover to limit the potential damage.

Control server example
-------------------

```python
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/auth')
def auth():
    return jsonify(
        tokens=[
            {
                'id': 'hello',
                'rates': [
                    {'increment': 1, 'value': 10, 'period': 'minute'}
                ]
            }
       ],
       upstreams=[
            {
                'url': 'http://localhost:5000/upstream',
                'rates': [
                    {'increment': 1, 'value': 2, 'period': 'minute'}
                 ]
            },
            {
                'url': 'http://localhost:5000/upstream2',
                'rates': [
                    {'increment': 1, 'value': 4, 'period': 'minute'}
                 ]
            }
       ])

@app.route('/upstream')
def upstream():
    return 'Upstream: Hello World!'

@app.route('/upstream2')
def upstream2():
    return 'Upstream2: Hello World!'

if __name__ == '__main__':
    app.run()
```

Installation
------------

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
