## Golang Lib for DevOps
  - metrics
  - monitoring
  - servers
  - global locking / paxos / raft
  - Cloud API clients


## Locking Server/PubSub/Service Discovery
  - GNats/Nats:
    - http://www.reddit.com/r/golang/comments/1oqqx7/gnatsd_from_apcera_a_high_performance_nats_server/
    - NATS does not have persistence, or transactions. It is more like a nervous system, and it will protect itself at all costs and does not have SPOFs. It does publish/subscribe, and distributed queues.
    - http://www.quora.com/Cloud-Foundry/Why-does-CloudFoundry-use-NATS-a-specially-written-messaging-system-whereas-OpenStack-uses-AMQP
    AMQP, and implementations like RabbitMQ, are enterprise messaging systems built with things like durability, transactions, and formal queues. NATS was designed and built to be like a dial-tone publish-subscribe service, something that is always on and available. However, NATS does not provide durability or transactions, and its queuing model is interest-based only. It also protects itself, the NATS service, at all costs, so that it can always be available. This forms a great base platform for building scalable and reliable distributed systems, but is probably not a good fit for the typical enterprise application.

  - Serf:
    - http://www.serfdom.io/
    - https://news.ycombinator.com/item?id=6600063



## Why Golang for DevOps?
  - [Go Language for Ops and Site Reliability Engineering](http://talks.golang.org/2013/go-sreops.slide)



This repository is supposed to work with [DirEnv](https://github.com/zimbatm/direnv). It will set the GOPATH to current directory and append the ./bin folder to your PATH variable.


<!-- PROJECTS_LIST_START -->
    *** GENERATED BY https://github.com/mindreframer/techwatcher (ruby _sh/pull golang-devops-stuff) *** 

    abh/geodns:
      DNS server with per-client targeted responses
       233 commits, last change: 2013-11-05 02:00:20, 236 stars, 18 forks

    adnaan/hamster:
      A back end as a service based on MongoDB
       43 commits, last change: 2013-09-22 02:54:00, 45 stars, 1 forks

    apcera/gnatsd:
      High Performance NATS Server
       269 commits, last change: 2013-11-18 16:03:16, 265 stars, 32 forks

    apcera/nats:
      NATS client for Go
       212 commits, last change: 2013-11-12 15:54:58, 57 stars, 7 forks

    benbjohnson/go-raft:
      A Go implementation of the Raft distributed consensus protocol.
       374 commits, last change: 2013-11-12 19:23:10, 511 stars, 50 forks

    bitly/google_auth_proxy:
      A reverse proxy that provides authentication using Google OAuth2
       19 commits, last change: 2013-10-24 10:42:28, 110 stars, 18 forks

    bitly/nsq:
      A realtime distributed messaging platform
       951 commits, last change: 2013-11-10 17:59:06, 1,682 stars, 145 forks

    buger/gor:
      HTTP traffic replay in real-time. Replay traffic from production to staging and dev environments.
       225 commits, last change: 2013-11-18 05:34:08, 947 stars, 50 forks

    BurntSushi/cmail:
      cmail runs a command and sends the output to your email address at certain intervals.
       8 commits, last change: , 3 stars, 0 forks

    ccding/go-stun:
      a go implementation of the STUN client (RFC 3489 and RFC 5389)
       4 commits, last change: 2013-08-17 16:10:34, 10 stars, 0 forks

    cloudflare/redoctober:
      Go server for two-man rule style file encryption and decryption.
       24 commits, last change: 2013-11-22 06:51:07, 202 stars, 7 forks

    cloudfoundry/gorouter:

       360 commits, last change: 2013-11-15 22:01:29, 92 stars, 28 forks

    cloudfoundry/gosigar:

       15 commits, last change: 2013-08-06 16:12:49, 35 stars, 16 forks

    cloudfoundry/hm9000:

       274 commits, last change: 2013-11-20 17:37:03, 10 stars, 2 forks

    cloudfoundry/yagnats:
      Yet Another Go NATS client
       56 commits, last change: 2013-11-14 12:01:27, 4 stars, 1 forks

    coreos/etcd:
      A highly-available key value store for shared configuration and service discovery
       1,018 commits, last change: 2013-11-20 10:35:55, 1,742 stars, 135 forks

    crowdmob/goamz:
      Fork of the GOAMZ version developed within Canonical with additional functionality with DynamoDB
       374 commits, last change: 2013-11-05 07:50:08, 59 stars, 35 forks

    dotcloud/docker:
      Docker - the open-source application container engine
       4,171 commits, last change: 2013-11-20 16:30:48, 7,382 stars, 935 forks

    efficient/epaxos:

       21 commits, last change: 2013-10-24 11:33:22, 130 stars, 8 forks

    errplane/errplane-go:
      Go library for metrics for Errplane
       52 commits, last change: 2013-08-21 14:51:17, 10 stars, 0 forks

    flynn/go-crypto-ssh:
      Forked from go.crypto as Flynn working copy until changes merged upstream
       5 commits, last change: 2013-10-22 14:14:21, 1 stars, 1 forks

    flynn/go-discover:
      Service discovery system written in Go
       43 commits, last change: 2013-11-03 20:08:58, 73 stars, 6 forks

    flynn/rpcplus:
      Go RPC plus streaming responses (forked from vitess)
       8 commits, last change: 2013-10-06 11:26:17, 11 stars, 1 forks

    globocom/gandalf:
      Gandalf is an API to manage git repositories.
       453 commits, last change: 2013-11-11 04:34:10, 101 stars, 16 forks

    globocom/tsuru:
      Open source Platform as a Service.
       6,153 commits, last change: 2013-11-19 15:04:59, 730 stars, 59 forks

    golang/groupcache:
      groupcache is a caching and cache-filling library, intended as a replacement for memcached in many cases.
       21 commits, last change: 2013-10-30 09:55:26, 2,364 stars, 202 forks

    hashicorp/serf:
      Service orchestration and management tool.
       537 commits, last change: 2013-11-20 08:54:21, 1,029 stars, 42 forks

    influxdb/influxdb:
      Scalable datastore for metrics, events, and real-time analytics
       446 commits, last change: 2013-11-20 12:37:01, 838 stars, 33 forks

    jondot/groundcontrol:
      Manage and monitor your Raspberry Pi with ease
       50 commits, last change: 2013-08-22 07:19:32, 619 stars, 42 forks

    jordansissel/lumberjack:
      An experiment to cut logs in preparation for processing elsewhere.
       529 commits, last change: 2013-11-20 12:46:07, 272 stars, 65 forks

    mindreframer/emtail:
      extract whitebox monitoring data from logs and insert into a timeseries database - mirror for https://code.google.com/p/emtail/
       273 commits, last change: , 0 stars, 0 forks

    mitchellh/packer:
      Packer is a tool for creating identical machine images for multiple platforms from a single source configuration.
       2,184 commits, last change: 2013-11-20 16:34:48, 1,556 stars, 205 forks

    mozilla-services/heka:
      Data collection and processing made easy.
       1,729 commits, last change: 2013-11-20 17:08:37, 816 stars, 68 forks

    necrogami/watchdog:
      Watchdog
       7 commits, last change: 2012-12-06 00:30:13, 1 stars, 1 forks

    nf/gohttptun:
      A tool to tunnel TCP over HTTP, written in Go
       20 commits, last change: 2013-09-22 17:01:00, 57 stars, 14 forks

    oleiade/trousseau:
      Networked and encrypted key-value database
       48 commits, last change: 2013-11-20 22:46:10, 163 stars, 9 forks

    petar/GoTeleport:
      Teleport Transport: End-to-end resilience to network outages
       6 commits, last change: 2013-08-30 10:54:44, 94 stars, 0 forks

    rackspace/gophercloud:
      A multi-cloud language binding for Go
       131 commits, last change: 2013-10-25 13:17:57, 173 stars, 12 forks

    rcrowley/go-metrics:
      Go port of Coda Hale's Metrics library
       147 commits, last change: 2013-10-31 15:13:26, 185 stars, 29 forks

    Sendhub/shipbuilder:
      The Open-source self-hosted Platform-as-a-Service written in Go
       135 commits, last change: 2013-10-21 17:24:27, 1 stars, 20 forks

    skydb/sky:
      Sky is an open source, behavioral analytics database.
       556 commits, last change: 2013-04-02 07:23:03, 273 stars, 26 forks

    skynetservices/skydns:
      DNS for skynet or any other service discovery
       71 commits, last change: 2013-11-17 05:55:38, 195 stars, 14 forks

    spf13/nitro:
      Quick and easy performance analyzer library for golang
       7 commits, last change: 2013-10-03 06:43:07, 65 stars, 3 forks

    uniqush/uniqush-push:
      Uniqush is a free and open source software which provides a unified push service for server-side notification to apps on mobile devices.
       406 commits, last change: 2013-11-07 12:23:32, 313 stars, 46 forks

    VividCortex/robustly:
      Run functions resiliently
       21 commits, last change: 2013-07-30 08:59:36, 44 stars, 1 forks

    xtaci/gonet:
      a game server skeleton with golang
       909 commits, last change: 2013-11-19 00:26:49, 100 stars, 41 forks
<!-- PROJECTS_LIST_END -->
