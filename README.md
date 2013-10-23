# Watchdog

## Overview

Watchdog is an implementation of gosigar for use of monitoring a server to a mongo database.

## To Compile
	
* you will need git, bzr and hg version controls installed.
* go get github.com/cloudfoundry/gosigar
* go get code.google.com/p/goconf/conf
* go get labix.org/v2/mgo
* git clone https://github.com/necrogami/watchdog
* cd watchdog
* go build watchdog.go
* set your config options
* issue ./watchdog


## Supported platforms

Currently targeting modern flavors of darwin and linux.

## License

Apache 2.0
