##Roadmap
As awesome as SkyDNS is in its current state we plan to further development with the following.

* Handle Priorities for explicitly supplied hosts/uuids. This brings up an interesting question, if you supply a host or uuid should it only return exact matches, or should it work similar to regions where your supplied value gets a higher priority than others.
* More comprehensive test suite
* Validation of services
* Benchmarks / Performance Improvements
* Priorities based on latency between the requested region, and the additional external regions, as well as load in the given regions
* Weights based on system load/memory availability on the given host, so that idle nodes receive a higher weight and therefore a larger percentage of the requests.
* Semantic version lookups (1.0 - returns any 1.0.x, 1.1.0 - Explicit 1.0.0, 1 - returns any 1.x.x), possibly priorities based on versions?
* Support for peers that don't participate in consensus
* Use the authorization secret for signing (TSIG) DNS responses
* Use DNSSEC
