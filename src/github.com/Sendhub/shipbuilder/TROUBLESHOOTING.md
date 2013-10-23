== howto: Troubleshoot Unresponsive ShipBuilder Dynos ==

Sometimes people deploy their app and it is not available or working as expected.  Here is a list of items to verify:

1. Does the app have a Procfile?  Verify the filename is correctly capitalized, "Procfile".

2. Has the domain name been added to the app?
    e.g.
    $ sb domains -amy-app
    sendhub.com

3. Is the domain name pointed at the ShipBuilder load-balancer(s)?
    e.g.:
    $ dig sendhub.com

    ; <<>> DiG 9.8.5-P1 <<>> sendhub.com
    ;; global options: +cmd
    ;; Got answer:
    ;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 32761
    ;; flags: qr rd ra; QUERY: 1, ANSWER: 2, AUTHORITY: 0, ADDITIONAL: 0

    ;; QUESTION SECTION:
    ;sendhub.com.                  IN        A

    ;; ANSWER SECTION:
    sendhub.com.        240        IN        A        107.20.144.118
    sendhub.com.        240        IN        A        54.227.243.28

    ;; Query time: 31 msec
    ;; SERVER: 208.67.222.222#53(208.67.222.222)
    ;; WHEN: Wed Oct 02 11:18:13 PDT 2013
    ;; MSG SIZE  rcvd: 60

4. Are any web dynos scaled up?
    $ sb ps -amy-app
    === web: dyno scale=1, actual=1
    web @ v1 [sb-node2a:10019]

5. Do the app's nodes have a green status on the HAProxy stats page?
    visit https://<your-load-balancer>/haproxy and locate your app on the page.  If one or more of the dynos have a red background there is a problem.

    One source of this kind of problem can be if the health check is failing.
    HAProxy issues "GET /" HTTP requests to each dyno to check that the index page of the app returns a 2xx or 3xx response status code.  If there is no index page (404), or if it has errors (5xx), or is password protected, then HAProxy will mark the dyno as down and not route requests to it.

