# Health Manager 9000

[![Build Status](https://travis-ci.org/cloudfoundry/hm-workspace.png)](https://travis-ci.org/cloudfoundry/hm-workspace)

HM 9000 is a rewrite of CloudFoundry's Health Manager.  HM 9000 is written in Golang and has a more modular architecture compared to the original ruby implementation.  HM 9000's dependencies are locked down in a separate repo, the [hm-workspace](https://github.com/cloudfoundry/hm-workspace).

There are several Go Packages in this repository, each with a comprehensive set of unit tests.  In addition there is an integration test that excercises the interactions between the various componetns.  What follows is a detailed breakdown.

## Relocation & Status Warning

cloudfoundry/hm9000 will eventually be promoted and move to cloudfoundry/health_manager.  This is the temporary home while it is under development.

hm9000 is not yet a complete replacement for health_manager -- we'll update this README when it's ready for primetime.

## HM9000's Architecture and High-Availability

HM9000 solves the high-availability problem by relying on ETCD, a robust high-availability store distributed across multiple nodes.  Individual HM9000 components are built to rely completely on the store for their knowledge of the world.  This removes the need for maintaining in-memory information and allows clarifies the relationship between the various components (all data must flow through the store).

To avoid the singleton problem, we will turn on multiple instances of each HM9000 component across multiple nodes.  These instances will vie for a lock in the high-availability store.  The instance that grabs the lock gets to run and is responsible for maintaining the lock.  Should that instance enter a bad state or die, the lock becomes available allowing another instance to pick up the slack.  Since all state is stored in the store, the backup component should be able to function independently of the failed component.

## Deployment

### Recovering from Failure

If HM9000 enters a bad state, the simplest solution - typically - is to delete the contents of the data store.

#### If you're running HM9000 against a single, local, etcd node:

    local  $ bosh_ssh hm9000_z1/0 #for example
    hm9000 $ sudo su -
    hm9000 $ monit stop etcd
    hm9000 $ mkdir /var/vcap/store/etcdstorage-bad #for example
    hm9000 $ mv /var/vcap/store/etcdstorage/* /var/vcap/store/etcdstorage-bad
    hm9000 $ monit start etcd

all the other components should recover gracefully.

The data files in etcdstorage-bad can then be downloaded and analyzed to try to understand what went wrong to put HM9000/etcd in a bad state.  If you don't think this is necessary: just blow away the contents of `/var/vcap/store/etcdstorage`.

#### If you're running HM9000 against an ETCD cluster:

1. `bosh ssh` into each ETCD node (`bosh vms` is your friend here.  We typically have `etcd_leader_z1/0`, `etcd_z1/0`, and `etcd_z2/0`)
2. `monit stop etcd` on all the boxes (better to stop them all simultaenously!)
3. Blow away (or move) ETCDs storage directory.  It's located under `/var/vcap/store`
4. `monit start etcd` on all the boxes
5. HM9000 should recover on its own.

### If Clustered ETCD can't handle the load

You can identify this scenario by monitoring the `DesiredStateSyncTimeInMilliseconds` and the `ActualStateListenerStoreUsagePercentage` metrics.  If the `DesiredStateSyncTimeInMilliseconds` exceeds ~5000 (5 seconds)  *and* the `ActualStateListenerStoreUsagePercentage` exceeds 50-70 (this is a percentage - so out of 100) then clustered ETCD *may* be unable to handle the load.

To resolve this, you'll need to pick one of the HM9000 nodes (`hm9000_z1/0` or `hm9000_z2/0`) and make it the solitary HM9000 node and point it at its local ETCD database.  Here's how - let's say we want to keep `hm9000_z1/0` around:

1. `bosh ssh` onto `hm9000_z2/0` and issue a `monit stop all`
2. `bosh ssh` onto `hm9000_z1/0` and issue a `monit stop all`
3. Edit `/var/vcap/jobs/hm9000/config/hm9000.json` and set `store_urls` to a single entry: `"store_urls": ["http://127.0.0.1:4001"],`
4. Now `monit start all` and tail `/var/vcap/sys/log/hm9000/hm9000_listener.stdout.log` you should see heartbeats come in and get succesfully saved to the store.
5. Eventually, `/var/vcap/packages/hm9000/hm9000 dump --config=/var/vcap/jobs/hm9000/config/hm9000.json` should report that the store is fresh (this is near the top of the output).



## Installing HM9000 locally

Assuming you have `go` v1.1.* installed:

1. Clone the HM-workspace:

        $ cd $HOME
        $ git clone https://github.com/cloudfoundry/hm-workspace
        $ export GOPATH=$HOME/hm-workspace
        $ export PATH=$HOME/hm-workspace/bin:$PATH
        $ cd hm-workspace
        $ git submodule update --init

2. Install `etcd`

        $ pushd ./src/github.com/coreos/etcd
        $ ./build
        $ mv etcd $GOPATH/bin/
        $ popd

3. Start `etcd`.  Open a new terminal session and:

        $ export PATH=$HOME/hm-workspace/bin:$PATH
        $ cd $HOME
        $ mkdir etcdstorage
        $ cd etcdstorage
        $ etcd

    `etcd` generates a number of files in CWD when run locally, hence `etcdstorage`

4. Running `hm9000`.  Back in the terminal you used to clone the hm-workspace you should be able to

        $ hm9000

    and get usage information

5. Running the tests
    
        $ go get github.com/onsi/ginkgo/ginkgo
        $ cd src/github.com/cloudfoundry/hm9000/
        $ ginkgo -r -skipMeasurements -race -failOnPending

    These tests will spin up their own instances of `etcd` as needed.  It shouldn't interfere with your long-running `etcd` server.

6. Updating hm9000.  You'll need to fetch the latest code *and* recompile the hm9000 binary:

        $ cd $GOPATH/src/github.com/cloudfoundry/hm9000
        $ git checkout master
        $ git pull
        $ go install .

## Running HM9000

`hm9000` requires a config file.  To get started:

    $ cd $GOPATH/src/github.com/cloudfoundry/hm9000
    $ cp ./config/default_config.json ./local_config.json
    $ vim ./local_config.json

You *must* specify a config file for all the `hm9000` commands.  You do this with (e.g.) `--config=./local_config.json`

### Fetching desired state

    hm9000 fetch_desired --config=./local_config.json

will connect to CC, fetch the desired state, put it in the store, then exit.  You can optionally pass `-poll` to fetch desired state periodically.

### Listening for actual state

    hm9000 listen --config=./local_config.json

will come up, listen to NATS for heartbeats, and put them in the store.

### Analyzing the desired and actual state

    hm9000 analyze --config=./local_config.json

will come up, compare the desired/actual state, and submit start and stop messages to the store.  You can optionally pass `-poll` to analyze periodically.

### Sending start and stop messages

    hm9000 send --config=./local_config.json

will come up, evaluate the pending starts and stops and publish them over NATS. You can optionally pass `-poll` to send messages periodically.

### Serving metrics (varz)

    hm9000 serve_metrics --config=./local_config.json

will come up, register with the [collector](http://github.com/cloudfoundry/collector) and provide a `/varz` end-point with data.

### Serving API

    hm9000 serve_api --config=./local_config.json

will come up and provide response to requests for `app.state` over NATS.

### Evacuator

    hm9000 evacuator --config=./local_config.json

will come up and listen for `droplet.exited` messages and send `start` messages for any evacuating droplets.  The `evacuator` is *not* necessary for deterministic evacuation but is provided for backward compatibility with old DEAs.  There is no harm in running the `evacuator` *during* deterministic evacuation.

### Shredder

    hm9000 shred --config=./local_config.json

The shredder will periodically (once per hour, by default) compact the store - removing any orphaned (empty) directories.  You can optionally pass `-poll` to send messages periodically.

### Dumping the contents of the store

    hm9000 dump --config=./local_config.json

will dump the entire contents of the store to stdout.  The output is structured in terms of apps and provides insight into the state of a cloud foundry installation.  If you want a raw dump of the store's contents pass the `--raw` flag.

`etcd` has a very simple [curlable API](http://github.com/coreos/etcd), which you can use in lieu of `dump`.

## HM9000 Config

HM9000 is configured using a JSON file.  Here are the available entries:

- `heartbeat_period_in_seconds`:  Almost all configurable time constants in HM9000's config are specified in terms of this one fundamental unit of time - the time interval between heartbeats in seconds.  This should match the value specified in the DEAs and is typically set to 10 seconds.


- `heartbeat_ttl_in_heartbeats`:  Incoming heartbeats are stored in the store with a TTL.  When this TTL expires the instane associated with the hearbeat is considered to have "gone missing".  This TTL is set to 3 heartbeat periods.

- `actual_freshness_ttl_in_heartbeats`:  This constant serves two purposes.  It is the TTL of the actual-state freshness key in the store.  The store's representation of the actual state is only considered fresh if the actual-state freshness key is present.  Moreover, the actual-state is fresh *only if* the actual-state freshness key has been present for *at least* `actual_freshness_ttl_in_heartbeats`.  This avoids the problem of having the first detected heartbeat render the entire actual-state fresh -- we must wait a reasonable period of time to hear from all DEAs before calling the actual-state fresh.  This TTL is set to 3 heartbeat periods

- `grace_period_in_heartbeats`:  A generic grace period used when scheduling messages.  For example, we delay start messages by this grace period to give a missing instance a chance to start up before sending a start message.  The grace period is set to 3 heartbeat periods.

- `desired_freshness_ttl_in_heartbeats`: The TTL of the desired-state freshness.  Set to 12 heartbeats.  The desired-state is considered stale if it has not been updated in 12 heartbeats.

- `store_max_concurrent_requests`:  The maximum number of concurrent requests that each component may make to the store.  Set to 30.

- `sender_message_limit`:  The maximum number of messages the sender should send per invocation.  Set to 30.


- `sender_polling_interval_in_heartbeats`:  The time period in heartbeat units between sender invocations when using `hm9000 send --poll`.  Set to 1.

- `sender_timeout_in_heartbeats`:  The timeout in heartbeat units for each sender invocation.  If an invocation of the sender takes longer than this the `hm9000 send --poll` command will fail.  Set to 10.

- `fetcher_polling_interval_in_heartbeats`:  The time period in heartbeat units between desired state fetcher invocations when using `hm9000 fetch_desired --poll`.  Set to 6.

- `fetcher_timeout_in_heartbeats`:  The timeout in heartbeat units for each desired state fetcher invocation.  If an invocation of the fetcher takes longer than this the `hm9000 fetch_desired --poll` command will fail.  Set to 60.

- `analyzer_polling_interval_in_heartbeats`:  The time period in heartbeat units between analyzer invocations when using `hm9000 analyze --poll`.  Set to 1.

- `analyzer_timeout_in_heartbeats`:  The timeout in heartbeat units for each analyzer invocation.  If an invocation of the analyzer takes longer than this the `hm9000 analyze --poll` command will fail.  Set to 10.

- `shredder_polling_interval_in_heartbeats`:  The time period in heartbeat units between shredder invocations when using `hm9000 shred --poll`.  Set to 360.

- `shredder_timeout_in_heartbeats`:  The timeout in heartbeat units for each shredder invocation.  If an invocation of the shredder takes longer than this the `hm9000 analyze --poll` command will fail.  Set to 6.

- `number_of_crashes_before_backoff_begins`: When an instance crashes HM9000 immediately restarts it.  If, however, the number of crashes exceeds this number HM9000 will apply an increasing delay to the restart.

- `starting_backoff_delay_in_heartbeats`: The initial delay (in heartbeat units) to apply to the restart message once an instance crashes more than `number_of_crashes_before_backoff_begins` times.

- `maximum_backoff_delay_in_heartbeats`: The restart delay associated with crashes doubles with each crash but is not allowed to exceed this value (in heartbeat units).


- `listener_heartbeat_sync_interval_in_milliseconds`: The listener aggregates heartbeats and flushes them to the store periodically with this interval.

- `store_heartbeat_cache_refresh_interval_in_milliseconds`: To improve performance when writing heartbeats, the store maintains a write-through cache of the store contents.  This cache is invalidated and refetched periodically with this interval.


- `cc_auth_user`: The user to use when authenticating with the CC desired state API.  Set by BOSH.

- `cc_auth_password`: The password to use when authenticating with the CC desired state API.  Set by BOSH.

- `cc_base_url`: The base url for the CC API.  Set by BOSH.

- `desired_state_batch_size`: The batch size when fetching desired state information from the CC.  Set to 500.

- `fetcher_network_timeout_in_seconds`:  Each API call to the CC must succeed within this timeout.  Set to 10 seconds.


- `store_schema_version`: The schema of the store.  HM9000 does not migrate the store, instead, if the store data format/layout changes and is no longer backward compatible the schema version must be bumped.

- `store_type`: One of `"etcd"` or `"ZooKeeper"`

- `store_urls`: An array of ETCD/ZooKeeper server URLs to connect to.

- `actual_freshness_key`: The key for the actual freshness in the store.  Set to `"/actual-fresh"`.

- `desired_freshness_key`: The key for the actual freshness in the store.  Set to `"/desired-fresh"`.


- `metrics_server_port`: The port on which to serve /varz metrics.  If set to 0 a random available port will be chosen.

- `metrics_server_user`: The username that must be used to authenticate with /varz.  If set to "" a random username will be generated.

- `metrics_server_password`: The password that must be used to authenticate with /varz.  If set to "" a random password will be generated.


- `log_level`: Must be one of `"INFO"` or `"DEBUG"`


- `sender_nats_start_subject`:  The NATS subject for HM9000's start messages.  Set to `"hm9000.start"`.

- `sender_nats_stop_subject`:  The NATS subject for HM9000's stop messages.  Set to `"hm9000.stop"`.

- `nats.host`: The NATS host.  Set by BOSH.

- `nats.port`: The NATS host.  Set by BOSH.

- `nats.user`: The user for NATS authentication.  Set by BOSH.

- `nats.password`: The password for NATS authentication.  Set by BOSH.

## HM9000 components

### `hm9000` (the top level) and `hm`

The top level is home to the `hm9000` CLI.  The `hm` package houses the CLI logic to keep the root directory cleaner.  The `hm` package is where the other components are instantiated, fed their dependencies, and executed.

### `actualstatelistener`

The `actualstatelistener` provides a simple listener daemon that monitors the `NATS` stream for app heartbeats.  It generates an entry in the `store` for each heartbeating app under `/actual/INSTANCE_GUID`.

It also maintains a `FreshnessTimestamp`  under `/actual-fresh` to allow other components to know whether or not they can trust the information under `/actual`

#### `desiredstatefetcher`

The `desiredstatefetcher` requests the desired state from the cloud controller.  It transparently manages fetching the authentication information over NATS and making batched http requests to the bulk api endpoint.

Desired state is stored under `/desired/APP_GUID-APP_VERSION

### `analyzer`

The `analyzer` comes up, analyzes the actual and desired state, and puts pending `start` and `stop` messages in the store.  If a `start` or `stop` message is *already* in the store, the analyzer will *not* override it.

### `sender`

The `sender` runs periodically and pulls pending messages out of the store and sends them over `NATS`.  The `sender` verifies that the messages should be sent before sending them (i.e. missing instances are still missing, extra instances are still extra, etc...) The `sender` is also responsible for throttling the rate at which messages are sent over NATS.

### `metricsserver`

The `metricsserver` registers with the CF collector and aggregates and provides metrics via a /varz end-point.  These are the available metrics:

- NumberOfAppsWithAllInstancesReporting: The number of desired applications for which all instances are reporting (the state of the instance is irrelevant: STARTING/RUNNING/CRASHED all count).
- NumberOfAppsWithMissingInstances: The number of desired applications for which an instance is missing (i.e. the instance is simply not heartbeating at all).
- NumberOfUndesiredRunningApps: The number of *undesired* applications with at least one instance reporting as STARTING or RUNNING.
- NumberOfRunningInstances: The number of instances in the STARTING or RUNNING state.
- NumberOfMissingIndices: The number of missing instances (these are instances that are desired but are simply not heartbeating at all).
- NumberOfCrashedInstances: The number of instances reporting as crashed.
- NumberOfCrashedIndices: The number of *indices* reporting as crashed.  Because of the restart policy an individual index may have very many crashes associated with it.

If either the actual state or desired state are not *fresh* all of these metrics will have the value `-1`.

### `apiserver`

The `apiserver` responds to NATS `app.state` messages and allow other CloudFoundry components to obtain information about arbitrary applications.

### `evacuator`

The `evacuator` responds to NATS `droplet.exited` messages.  If an app exists because it is EVACUATING the `evacuator` sends a `start` message over NATS.  The `evacuator` is not necessary during deterministic evacuations but is provided to maintain backward compatibility with older DEAs.

### `shredder`

The `shredder` prunes old/crufty/unnecessary data from the store.  This includes pruning old schema versions of the store.

## Support Packages

### `config`

`config` parses the `config.json` configuration.  Components are typically given an instance of `config` by the `hm` CLI.

### `helpers`

`helpers` contains a number of support utilities.

#### `httpclient`

A trivial wrapper around `net/http` that improves testability of http requests.

#### `logger`

Provides a (sys)logger.  Eventually this will use steno to perform logging.

#### `metricsaccountant`

Supports metrics tracking.  Used by the `metricsserver` and components that post metrics.

#### `timeprovider`

Provides a `TimeProvider`.  Useful for injecting time dependencies in tests.

#### `workerpool`

Provides a worker pool with a configurable pool size.  Work scheduled on the pool will run concurrently, but no more `poolSize` workers can be running at any given moment.

### `models`

`models` encapsulates the various JSON structs that are sent/received over NATS/HTTP.  Simple serializing/deserializing behavior is attached to these structs.

### `store`

`store` sits on top of the lower-level `storeadapter` and provides the various hm9000 components with high-level access to the store (components speak to the `store` about setting and fetching models instead of the lower-level `StoreNode` defined inthe `storeadapter`).

### `storeadapter`

The `storeadapter` is an generalized client for connecting to a Zookeeper/ETCD-like high availability store.  Writes are performed concurrently for optimal performance.

## Test Support Packages (under testhelpers)

`testhelpers` contains a (large) number of test support packages.  These range from simple fakes to comprehensive libraries used for faking out other CloudFoundry components (e.g. heartbeating DEAs) in integration tests.

### Fakes

#### `fakelogger`

Provides a fake implementation of the `helpers/logger` interface

#### `faketimeprovider`

Provides a fake implementation of the `helpers/timeprovider` interface.  Useful for injecting time dependency in test.

#### `fakehttpclient`

Provides a fake implementation of the `helpers/httpclient` interface that allows tests to have fine-grained control over the http request/response lifecycle.

#### `fakemetricsaccountant`

Provides a fake implementation of the `helpers/metricsaccountant` interface that allows test to make assertions on metrics tracking.

#### `fakestoreadapter`

Provides a fake in-memory implementation of the `storeadapter` to allow for unit tests that do not need to spin up a database.

### Fixtures & Misc.

#### `app`

`app` is a simple domain object that encapsulates a running CloudFoundry app.

The `app` package can be used to generate self-consistent data structures (heartbeats, desired state).  These data structures are then passed into the other test helpers to simulate a CloudFoundry eco-system.

Think of `app` as your source of fixture test data.  It's intended to be used in integration tests *and* unit tests.

Some brief documentation -- look at the code and tests for more:

```go
//get a new fixture app, this will generate appropriate
//random APP and VERSION GUIDs
app := NewApp()

//Get the desired state for the app.  This can be passed into
//the desired state server to simulate the APP's presence in 
//the CC's DB.  By default the app is staged and started, to change
//this, modify the return value.
desiredState := app.DesiredState(NUMBER_OF_DESIRED_INSTANCES)

//get an instance at index 0.  this getter will lazily create and memoize
//instances and populate them with an INSTANCE_GUID and the correct
//INDEX.
instance0 := app.InstanceAtIndex(0)

//generate a heartbeat for the app.
//note that the INSTANCE_GUID associated with the instance at index 0 will
//match that provided by app.InstanceAtIndex(0)
app.Heartbeat(NUMBER_OF_HEARTBEATING_INSTANCES)
```

#### `custommatchers`

Provides a collection of custom Gomega matchers.

### Infrastructure Helpers


#### `startstoplistener`

Listens on the NATS bus for `health.start` and `health.stop` messages.  It parses these messages and makes them available via a simple interface.  Useful for testing that messages are sent by the health manager appropriately.

#### `desiredstateserver`

Brings up an in-process http server that mimics the CC's bulk endpoints (including authentication via NATS and pagination).

#### `natsrunner`

Brings up and manages the lifecycle of a live NATS server.  After bringing the server up it provides a fully configured cfmessagebus object that you can pass to your test subjects.

#### `storerunner`

Brings up and manages the lifecycle of a live ETCD/ZooKeeper server cluster.

## The MCAT

The MCAT is as HM9000's integration test suite.  It tests HM9000 by providing it with inputs (desired state, actual state heartbeats, and time) and asserting on its outputs (start and stop messages and api/metrics endpoints).

In addition to the MCAT there is a performance-measuring test suite at [https://github.com/pivotal-cf-experimental/hmperformance](https://github.com/pivotal-cf-experimental/hmperformance).
