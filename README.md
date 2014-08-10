## Go Share

```ASCII
                        __
   ____ _____     _____/ /_  ____ _________
  / __ `/ __ \   / ___/ __ \/ __ `/ ___/ _ \
 / /_/ / /_/ /  (__  ) / / / /_/ / /  /  __/
 \__, /\____/  /____/_/ /_/\__,_/_/   \___/
/____/

```

[Tasks in Queue at Trello Board](https://trello.com/b/ZjDMRGQN/goshare)

distributed under [MIT License](http://opensource.org/licenses/MIT)

#### Go Share any data among the nodes. Over HTTP or ZeroMQ.

* GOShare eases up communication over HTTP GET param based interaction.
* ZeroMQ REQ/REP based synchronous communication model.

it's "go get"-able

``` go get "github.com/abhishekkr/goshare" ```

***

#### Make Distributable Binary

```bash
./go-tasks.sh bin
```

This will create two distributable binaries *./bin/goshare_service* & *./bin/goshare_daemon*. Here *./bin/goshare_service* works as shown in README's 'Tryout' section.

Whereas *./bin/goshare_daemon* can be used as a system service daemon, with following command line flags (along with flags mentioned in 'Tryout' section for ports and db-path)
>
> * start: ``` ./bin/goshare_daemon -daemon=start ```
> * stop: ``` ./bin/goshare_daemon -daemon=stop ```
> * status: ``` ./bin/goshare_daemon -daemon=status ```
>
> *this dumps daemon's current status to /tmp/goshare_daemon.status and pid to /tmp/goshare_daemon.pid*
> *the status and pid file path can be changed with flags '-daemon-log=<path>' & '-daemon-pid=<path>' respectively*
>


#### Tryout:

```Shell
 go run zxtra/goshare_daemon.go -dbpath=/tmp/GOTSDB
```

By default it runs HTTP daemon at port 9999 and ZeroMQ daemon at 9797/9898,
make it run on another port using following required flags

```Shell
 go run zxtra/goshare_daemon.go -dbpath=/tmp/GOTSDB -port=8080 -req-port=8000 -rep-port=8001
```

```ASCII
  Dummy Clients Using It

  * go run zxtra/gohttp_client.go

  * go run zxtra/go0mq_client.go


  for custom Port: 8080 for HTTP; Port: 8000/8001 for ZeroMQ

  * go run zxtra/gohttp_client.go -port=8080

  * go run zxtra/go0mq_client.go -req-port=8000 -rep-port=8001
```

>
> To utilize it "zxtra/gohttp_client.go" and "zxtra/go0mq_client.go" can be referred on how to utilize capabilities of GoShare.
>

***

#### Structure:

> "goshare"'s methods to adapt these in your code:
>
> * GoShare() : it runs HTTP and ZeroMQ daemon in parallel goroutines
> > has optional flags customization of:
> > * dbpath: path for LevelDB (default: /tmp/GO.DB)
> > * port: port to bind HTTP daemon (default: 9999)
> > * req-port, rep-port: ports to bind ZeroMQ REQ/REP daemon (default: 9797, 9898)
>
> * GoShareHTTP(&lt;levigo DB handle&gt;, &lt;http port as int&gt;) : it runs HTTP daemon
>
> * GoShareZMQ(&lt;levigo DB handle&gt;, &lt;req-port as int&gt;, &lt;rep-port as int&gt;) : it runs ZMQ daemon
>

***

Now visit the the link asked by it and get the help page.

##### Dependency
* [go lang](http://golang.org/doc/install) (obviously, the heart and soul of the app)
* [leveldb](http://en.wikipedia.org/wiki/LevelDB) (we are using for datastore, it's awesome)
* [levigo](https://github.com/jmhodges/levigo/blob/master/README.md) (the go library utilized to access leveldb)
* [zeroMQ](http://zeromq.org/) (the supercharged Sockets giving REQuest/REPly power)
* [gozmq](https://github.com/alecthomas/gozmq) GoLang ZeroMQ Bindings used here
* [levigoNS](https://github.com/abhishekkr/levigoNS) NameSpace KeyVal capabilities around leveldb via levigo
* [levigoTSDS](https://github.com/abhishekkr/levigoTSDS) TimeSeries KeyVal capabilties around leveldb via levigoNS
* [gol](https://github.com/abhishekkr/gol) Set of common utility functionalities

[![baby-gopher](https://raw2.github.com/drnic/babygopher-site/gh-pages/images/babygopher-badge.png)](http://www.babygopher.org)
