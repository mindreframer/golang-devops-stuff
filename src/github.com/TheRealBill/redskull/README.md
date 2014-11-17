# What Is Red Skull?

Red Skull is a Sentinel management system. It is designed to run on each
sentinel node you operate and provide a single, yet distributed,
mechanism for managing Sentinel as well as interacting with it.

# Overview

Written in Go, Red Skull runs on each Sentinel and bootstraps itself
from that Sentinel's configuration file. It will then interrogate any
`known-sentinel` directives as well as run `setinel sentinels <name> for
each pod found in the config file.  It essentially crawls through your
Sentinel constellation and discovers all sentinels, masters, and slaves.

It then provides a decent web interface for viewing and managing your
sentinels, and by proxy the Redis instances under management. It
introduces some new concepts/terminology and these will be explained in
the documentation tree.

In addition to the front end Red Skull provides an HTTP/JSON REST-*like*
interface for interacting with programmaticly. Adding the redis Sentinel
API as another interface is planned as well.


# Current State

The initial import is of the base working code. It still likely has many
bugs as it is the result of only ~2.5 total weeks of effort and there
are still much error handling to be written.  That said, the base
functionality is there and working.

The initial effort after import will be a focus on documenting Red
Skull.  Primarily how to install and use it; its design, goals, and
contribution guidelines; and the direction and needs for it's
advancement.

Can you use it for "production use". Yes. Will it destroy your setup?
Not likely. 

Most of the things you can do in the web UI are also available in the
JSON+HTTP API but there may be some new functionality I've not yet added
to the API.


# Requirements

As RS is written in Go you need Go installed. Once cloned, you will need to
install a few dependencies:

* go get "github.com/kelseyhightower/envconfig"
* go get "github.com/therealbill/airbrake-go"
* go get "github.com/therealbill/libredis/client"
* go get "github.com/therealbill/libredis/info"
* go get "github.com/zenazn/goji"

Then you can execute `go build` in the root of the repo

# Installation

Assuming you have Git and Go (sounds like a techie oriented convenience
store - "the Git and Go") installed, installing Red Skull is fairly
simple. The dependencies are listed in the Godeps file. If you have/use
[gpm](https://github.com/pote/gpm) (a Go dependency manager), you can do
the following:

```shell
go get github.com/therealbill/redskull 
cd $GOPATH/src/github.com/therealbill/redskull 
gpm install 
go build
./redskull
```

And, assuming you have a sentinel config at /etc/redis/sentinel.conf it
will be up and running on localhost port 8000.

If you don't use gpm the following should work reasonably well:
```
go get github.com/therealbill/redskull 
cd $GOPATH/src/github.com/therealbill/redskull 
for x in `cat Godeps`; do
go get $x 
go build
./redskull
```

# Running Red Skull

Red Skull expects to find the sentinel config file in
/etc/redis/sentinel.conf.  You can, however, alter this by the setting
the environment variable REDSKULL_SENTINELCONFIGFILE.

RS currently expects the html directory to be in the same location as
the binary. For example you can do create  adirectory named
`/usr/local/redskull`, place the redskull binary in it, and copy the
html directory to it, then launch `./redskull` and it should work.
You'll find it running on port 8000

I'll be making locations configurable soon.


# Calling the API

Err, for now look in main.go to see the URLs and whether you need to do
a GET, PUT, DEL, or POST for that call. Most of it is pretty simple.
I've just not documented it yet as I prefer to do it once things
stabilize. If you want to help get that jumpstarted pull requests are
welcome. :)


Can you use it for "production use"? Yes. Will it destroy your setup?
Not likely. It only executes read-only commands unless you click the
button to make a change.
