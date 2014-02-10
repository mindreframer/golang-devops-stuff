## skydnsctl - cli tool for interacting with SkyDNS

#### Commands
* add
* list
* update
* delete


### Connect to your SkydNS HTTP endpoint
To connect to the SkyDNS HTTP endpoint for issuing commands set the environment 
variable `SKYDNS` or you can run the cli app with the `--host` flag.

```bash
export SKYDNS="http://localhost:8080"
# OR
skydnsctl --host "http://localhost:8080"
```

### SkyDNS DNS endpoint

For the DNS discovery port 53 is assumed on the URL mentioned in `--host` or in 
the SKYDNS environment variable. This can overrulled with `--dnsport` or the environment
variable SKYDNS_DNSPORT.

The default domain used for DNS queries is `skydns.local`, but this can be overruled with the
environment variable SKYDNS_DNSDOMAIN or the option `--dnsdomain`.

#### Add a new service

```bash
skydnsctl add  1001 '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web
1.site.com","Port":9000,"TTL":1000}'
1001 added to skydns
```

#### Get an existing service by UUID

```bash
skydnsctl 1001

UUID: 1001
Name: TestService
Host: web1.site.com
Port: 9000
Environment: Production
Region: Test
Version: 1.0.0

TTL 492
Remaining TTL: 492
```

#### Get all services

```bash
skydnsctl
UUID: 1004
Name: TestService
Host: web4.site.com
Port: 80
Environment: Production
Region: West
Version: 1.0.0

TTL 141
Remaining TTL: 141

----
```

#### Get an existing service with json output

```bash
skydnsctl --json 1001
{"UUID":"1001","Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com","Port":9000,"TTL":987,"Expires":"2014-01-17T23:09:19.827085688-08:00"}
```

#### Update an existing service

```bash
skydnsctl update 1001 3000
1001 ttl updated to 3000
```

#### Delete an existing service

```bash
skydnsctl delete 1001
1001 removed from skydns
```
