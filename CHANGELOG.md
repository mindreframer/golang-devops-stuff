# 0.8.0 (3/18/2014)

* Moved to the [gollector\_metrics](https://github.com/gollector/gollector_metrics) library.

# 0.7.1 (2/8/2014)

* Corrected a bug in fs\_usage percentage: was not accounting for the reserved
  space root gets.

# 0.7.0 (2/7/2014)

* Graphite support! Via a bridge called `gollector-graphite`, emits stats to
  graphite collected from gollector.

# 0.6.1 (2/2/2014)

* Fix a bug where json\_poll would keep sockets open forever, causing FD
  leakage.

# 0.6.0 (1/13/2014)

Three new plugins:

* `socket_usage`: report on the number of sockets open for a given protocol
* `process_count`: count the number of command lines that are running
* `process_mem_usage`: count the amount of memory a given command line is using across all processes

# 0.5.0 (1/4/2014)

Many Refactors. All Bugs & Features (below) are covered in the wiki. This is
the first release with the name "Gollector", as well.

* `mem_usage` now reports swap totals.
* LogLevel in the configuration is now properly used.
* `json_poll` now can use a unix socket.

# Rename (11/11/2013)

Cirgonus is now known as Gollector! All naughty bits have been changed to
reflect this, including the name of the repository.

Since there are no new features, no version has been created.

# 0.4.0 (10/24/2013)

All of these features are covered in the README documentation.

* JSON HTTP polling plugin allows cirgonus to periodically poll a resource for
  injectable metrics.
* Cirgonus can now take conf.d style configuration directories which makes it
  easier to drive with configuration management.

# 0.3.0 (10/11/2013)

All of these features are covered in the README documentation.

* cstat is now able to query multiple metrics at once from each host.
* The `fs_usage` plugin reports on usage stats for a mountpoint, and its read-only status.

# 0.2.0 (10/9/2013)

All of these features are covered in the README documentation.

* Cirgonus no longer polls on each hit -- it does so on a tick value then
  serves requests from cache. You can adjust the frequency at which it polls by
  tweaking the "PollInterval" json configuration, which defaults to 60 seconds
  for `cirgonus generate`.
* Now logging to syslog -- you can adjust the facility at which it logs to by
  tweaking `Facility` in the json configuration which defaults to `daemon` and
  `LogLevel` for scoping messages, which defaults to `info`.
* Result publishing lets you push metrics to cirgonus which will then be picked
  up by circonus as a metric -- great for custom tooling!
* Makefile now to build releases and copies of `cirgonus` and `cstat`

# 0.1.0 (09/22/2013)

* First release!
