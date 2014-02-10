#SkyDNS [![Build Status](https://travis-ci.org/skynetservices/skydns.png)](https://travis-ci.org/skynetservices/skydns)
*Version 0.2.0*

SkyDNS is a distributed service for announcement and discovery of services. It
leverages Raft for high-availability and consensus, and utilizes DNS queries
to discover available services. This is done by leveraging SRV records in DNS,
with special meaning given to subdomains, priorities and weights.

SkyDNS will also act as a forwarding DNS proxy, so that you can set your SkyDNS
instance as the primary DNS service in /etc/resolv.conf and SkyDNS will forward
and proxy requests for which it is not authoritative.

Besides serving SRV records, which include *all* the information you need to
connect to your service, SkyDNS will also return A records. This is useful if
you already know what port a particular service is using, and you just want a
list of IP addresses with known running instances.

[Announcement Blog Post](http://blog.gopheracademy.com/skydns)

##Setup / Install

Compile SkyDNS, and execute it

`go get -d -v ./... && go build -v ./...`

`./skydns`

Which takes the following flags
- -domain - This is the domain requests are anchored to and should be appended to all requests (Defaults to: skydns.local)
- -http - This is the HTTP ip:port to listen on for API request (Defaults to: 127.0.0.1:8080)
- -dns - This is the ip:port to listen on for DNS requests (Defaults to: 127.0.0.1:53)
- -data - Directory that Raft logs will be stored in (Defaults to: ./data)
- -join - When running a cluster of SkyDNS servers as recommended, you'll need to supply followers with where the other members can be found, this can be any member or a comma separated list of members. It does not have to be the leader. Any non-leader you join will redirect you to the leader automatically.
- -discover - This flag can be used in place of explicitly supplying cluster members via the -join flag. It performs a DNS lookup using the hosts DNS server for NS records associated with the -domain flag to find the SkyDNS instances.
- -metricsToStdErr - When this flag is set to true, metrics will be periodically written to standard error
- -graphiteServer - When this flag is set to a Graphite Server URL:PORT, metrics will be posted to a graphite server
- -stathatUser - When this flag is set to a valid StatHat user, metrics will be posted to that user's StatHat account periodically
- -secret - When this variable is set, the HTTP api will require an authorization header that matches the secret passed to skydns when it starts  
- -nameserver - Nameserver address to forward (non-local) queries to e.g. "8.8.8.8:53,8.8.4.4:53", in other words an IP:PORT, where multiple nameservers maybe listed separated by a comma "`,`". If this list is empty (""),
SkyDNS will parse /etc/resolv.conf and will use the nameservers listed there.

##API
### Service Announcements
You announce your service by submitting JSON over HTTP to SkyDNS with information about your service.
This information will then be available for queries either via DNS or HTTP.

When providing information you will need to fill out the following values. Note you are free to use
whatever you like, so take the following list as a guide only.

* Name - The name of your service, e.g., "rails", "web" or anything else you like
* Version - A version string, note the dots in this string are translated to hyphens when
    querying via the DNS
* Environment - Can be something as "production" or "testing"
* Region - Where do these hosts live, e.g. "east", "west" or even "test"
* Host, Port and TTL - Denote the actuals hosts and how long (TTL) this information is valid.

When queried SkyDNS will return records containing these elements in the following
order:

    <uuid>.<host>.<region>.<version>.<service>.<environment>.skydns.local

Where `<uuid>` is the identifier used when registering this host and service. And also
note the `<service>` corresponds with the Name given above.

Note some of these elements may contain a wildcard or be left out completely,
see the section named "Wildcards" below for more information.

#### Without Shared Secret 
`curl -X PUT -L http://localhost:8080/skydns/services/1001 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com","Port":9000,"TTL":10}'`

#### With Shared Secret 
You have the ability to use a shared secret with SkyDns. To take advantage of the shared secret you would start skydns with the -secret=<secretString> flag.
`curl -X PUT -H "Authorization mysupersecretsharedsecret" -L http://localhost:8080/skydns/services/1001 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com","Port":9000,"TTL":10}'`

If unsuccessful you should receive an HTTP status code of: **403 Forbidden**

#### Result 

If successful you should receive an HTTP status code of: **201 Created**

If a service with this UUID already exists you will receive back an HTTP status
code of: **409 Conflict**

SkyDNS will now have an entry for your service that will live for the number
of seconds supplied in your TTL (10 seconds in our example), unless you send a
heartbeat to update the TTL.

Note that instead of a hostname you can also use an IP address (IPv4 or IPV6),
in that case SkyDNS will make up an hostname that is used in the SRV record
(defaults to UUID.skydns.local) and adds the IP adress as an A or AAAAA record
in the additional section for this hostname.

### Heartbeat / Keep alive
SkyDNS requires that services submit an HTTP request to update their TTL within
the TTL they last supplied. If the service fails to do so within this timeframe
SkyDNS will expire the service automatically. This will allow for nodes to fail
and DNS to reflect this quickly.

You can update your TTL by sending an HTTP request to SkyDNS with an updated
TTL, it can be the same as before to allow it to live for another 10s, or it can
be adjusted to a shorter or longer duration.

`curl -X PATCH -L http://localhost:8080/skydns/services/1001 -d '{"TTL":10}'`

### Service Removal
If you wish to remove your service from SkyDNS for any reason without waiting for the TTL to expire, you simply send an HTTP DELETE.

`curl -X DELETE -L http://localhost:8080/skydns/services/1001`

### Retrieve Service Info via API
Currently you may only retrieve a service's info by UUID of the service, in the
future we may implement querying of the services similar to the DNS interface.

`curl -X GET -L http://localhost:8080/skydns/services/1001`

### Call backs
Registering a call back is similar to registering a service. A service that
registers a call back will receive an HTTP request. Every time something changes
in the service: the callback is executed, currently they are called when the
service is deleted.

`curl -X PUT -L http://localhost:8080/skydns/callbacks/1001 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com",Reply:"web2.example.nl","Port":5441}'`

This will result in the call back being sent to `web2.example.nl` on port 5441. The
callback itself will be a HTTP DELETE:

`curl -X DELETE -L http://web2.example.nl:5441/skydns/callbacks/1001 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com"}'`

##Discovery (DNS)
You can find services by querying SkyDNS via any DNS client or utility. It uses a known domain syntax with wildcards to find matching services.

Priorities and Weights are based on the requested Region, as well as how many nodes are available matching the current request in the given region.

###Domain Format
The domain syntax when querying follows a pattern where the right
most positions are more generic, than the subdomains to their left:
*\<uuid\>.\<host\>.\<region\>.\<version\>.\<service\>.\<environment\>.skydns.local*. 
This allows for you to supply only the positions you care about:

- authservice.production.skydns.local - For instance would return all services with the name AuthService in the production environment, regardless of the Version, Region, or Host
- 1-0-0.authservice.production.skydns.local - Is the same as above but restricting it to only version 1.0.0
- east.1-0-0.authservice.production.skydns.local - Would add the restriction that the services must be running in the East region

#### Wildcards

In addition to only needing to specify as much of the domain as required for the granularity level you're looking for, you may also supply the wildcard `*` in any of the positions.

- east.*.*.production.skydns.local - Would return all services in the East region, that are a part of the production environment.

###Examples

Let's take a look at some results. First we need to add a few services so we have services to query against.

	// Service 1001 (East Region)
	curl -X PUT -L http://localhost:8080/skydns/services/1001 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"East","Host":"web1.site.com","Port":80,"TTL":4000}'
	
	// Service 1002 (East Region)
	curl -X PUT -L http://localhost:8080/skydns/services/1002 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"East","Host":"web2.site.com","Port":8080,"TTL":4000}'
	
	// Service 1003 (West Region)
	curl -X PUT -L http://localhost:8080/skydns/services/1003 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"West","Host":"web3.site.com","Port":80,"TTL":4000}'
	
	// Service 1004 (West Region)
	curl -X PUT -L http://localhost:8080/skydns/services/1004 -d '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"West","Host":"web4.site.com","Port":80,"TTL":4000}'

Now we can try some of our example DNS lookups:
#####All services in the Production Environment
`dig @localhost production.skydns.local SRV`

	;; QUESTION SECTION:
	;production.skydns.local.			IN	SRV

	;; ANSWER SECTION:
	production.skydns.local.		629		IN	SRV	10 20 80   web1.site.com.
	production.skydns.local.		3979	IN	SRV	10 20 8080 web2.site.com.
	production.skydns.local.		3629	IN	SRV	10 20 9000 server24.
	production.skydns.local.		3985	IN	SRV	10 20 80   web3.site.com.
	production.skydns.local.		3990	IN	SRV	10 20 80   web4.site.com.

#####All TestService instances in Production Environment
`dig @localhost testservice.production.skydns.local SRV`

	;; QUESTION SECTION:
	;testservice.production.skydns.local.		IN	SRV

	;; ANSWER SECTION:
	testservice.production.skydns.local.	615		IN	SRV	10 20 80   web1.site.com.
	testservice.production.skydns.local.	3966	IN	SRV	10 20 8080 web2.site.com.
	testservice.production.skydns.local.	3615	IN	SRV	10 20 9000 server24.
	testservice.production.skydns.local.	3972	IN	SRV	10 20 80   web3.site.com.
	testservice.production.skydns.local.	3976	IN	SRV	10 20 80   web4.site.com.

#####All TestService v1.0.0 Instances in Production Environment
`dig @localhost 1-0-0.testservice.production.skydns.local SRV`

	;; QUESTION SECTION:
	;1-0-0.testservice.production.skydns.local.	IN	SRV

	;; ANSWER SECTION:
	1-0-0.testservice.production.skydns.local. 600  IN	SRV	10 20 80   web1.site.com.
	1-0-0.testservice.production.skydns.local. 3950 IN	SRV	10 20 8080 web2.site.com.
	1-0-0.testservice.production.skydns.local. 3600 IN	SRV	10 20 9000 server24.
	1-0-0.testservice.production.skydns.local. 3956 IN	SRV	10 20 80   web3.site.com.
	1-0-0.testservice.production.skydns.local. 3961 IN	SRV	10 20 80   web4.site.com.

#####All TestService Instances at any version, within the East region
`dig @localhost east.*.testservice.production.skydns.local SRV`

This is where we've changed things up a bit, notice we used the "*" wildcard for
version so we get any version, and because we've supplied an explicit region
that we're looking for we get that as the highest DNS priority, with the weight
being distributed evenly, then all of our West instances still show up for
fail-over, but with a higher Priority.

	;; QUESTION SECTION:
	;east.*.testservice.production.skydns.local. IN	SRV

	;; ANSWER SECTION:
	east.*.testservice.production.skydns.local. 531  IN SRV	10 50 80   web1.site.com.
	east.*.testservice.production.skydns.local. 3881 IN SRV	10 50 8080 web2.site.com.
	east.*.testservice.production.skydns.local. 3531 IN SRV	20 33 9000 server24.
	east.*.testservice.production.skydns.local. 3887 IN SRV	20 33 80   web3.site.com.
	east.*.testservice.production.skydns.local. 3892 IN SRV	20 33 80   web4.site.com.


####A Records
To return A records, simply run a normal DNS query for a service matching the above patterns.

Let's add some web servers to SkyDNS:

	curl -X PUT -L http://localhost:8080/skydns/services/1011 -d '{"Name":"rails","Version":"1.0.0","Environment":"Production","Region":"East","Host":"127.0.0.10","Port":80,"TTL":400000}'
	curl -X PUT -L http://localhost:8080/skydns/services/1012 -d '{"Name":"rails","Version":"1.0.0","Environment":"Production","Region":"East","Host":"127.0.0.11","Port":80,"TTL":400000}'
	curl -X PUT -L http://localhost:8080/skydns/services/1013 -d '{"Name":"rails","Version":"1.0.0","Environment":"Production","Region":"West","Host":"127.0.0.12","Port":80,"TTL":400000}'
	curl -X PUT -L http://localhost:8080/skydns/services/1014 -d '{"Name":"rails","Version":"1.0.0","Environment":"Production","Region":"West","Host":"127.0.0.13","Port":80,"TTL":400000}'

Now do a normal DNS query:
`dig rails.production.skydns.local`

	;; QUESTION SECTION:
	;rails.production.skydns.local.	IN	A

	;; ANSWER SECTION:
	rails.production.skydns.local. 399918 IN A	127.0.0.10
	rails.production.skydns.local. 399918 IN A	127.0.0.11
	rails.production.skydns.local. 399918 IN A	127.0.0.12
	rails.production.skydns.local. 399919 IN A	127.0.0.13

Now you have a list of all known IP Addresses registered running the `rails`
service name. Because we're returning A records and not SRV records, there
are no ports listed, so this is only useful when you're querying for services
running on ports known to you in advance. Notice, we didn't specify version or
region, but we could have.

####DNS Forwarding

By specifying `-nameserver="8.8.8.8:53,8.8.4.4:53` on the `skydns` command line,
you create a DNS forwarding proxy. In this case it round robins between the two
nameserver IPs mentioned on the command line.

Requests for which SkyDNS isn't authoritative
will be forwarded and proxied back to the client. This means that you can set
SkyDNS as the primary DNS server in `/etc/resolv.conf` and use it for both service
discovery and normal DNS operations. 

*Please test this before relying on it in production, as there may be edge cases that don't work as planned.*

## License
The MIT License (MIT)

Copyright © 2014 The SkyDNS Authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
