After further investigation, I've discovered a way for us to simplify our approach to High-availability apps with DNS.

Previously I'd instructed you to add two A name (IP address) records for each subdomain you wanted to have available in your app; one for the IP of sb-lb1a and another for sb-lb1b.

Example DNS rules with current strategy:

    A sb-lb1a.sendhub.com -> 1.2.3.4
    A sb-lb1b.sendhub.com -> 1.2.3.5
    A staging1.sendhub.com -> 1.2.3.4
    A staging1.sendhub.com -> 1.2.3.5
    A staging4.sendhub.com -> 1.2.3.4
    A staging4.sendhub.com -> 1.2.3.5

    A www-staging1.sendhub.com -> 1.2.3.4
    A www-staging1.sendhub.com -> 1.2.3.5
    A www-staging4.sendhub.com -> 1.2.3.4
    A www-staging4.sendhub.com -> 1.2.3.5


This led to a confusing page of DNS rules; two A records for each subdomain entry.  With IP addresses sprinkled everywhere.  Certainly not helpful for troubleshooting DNS issues.

Subsequently, I've come up with an alternate solution which I think is easier and more elegant.  Put the duplicate IP addresses on a single subdomain in the form of two A records (see "sb-lb1.sendhub.com" below), and then use a single CNAME record for all subsequent application subdomains.

Equivalent DNS rules with new strategy:

    A sb-lb1a.sendhub.com -> 1.2.3.4
    A sb-lb1b.sendhub.com -> 1.2.3.5
    A sb-lb1.sendhub.com -> 1.2.3.4
    A sb-lb1.sendhub.com -> 1.2.3.5
    CNAME staging1.sendhub.com -> sb-lb1.sendhub.com
    CNAME staging4.sendhub.com -> sb-lb1.sendhub.com
    CNAME www-staging1.sendhub.com -> sb-lb1.sendhub.com
    CNAME www-staging4.sendhub.com -> sb-lb1.sendhub.com


For reference: From now on after you create a subdomain, you can verify that it's correct by running `dig <NewSubdomainHere>.sendhub.com` and ensuring that the structure matches the example below, with the new name being a CNAME pointed at sb-lb1.sendhub.com, and sb-lb1.sendhub.com pointing at the load-balancer IPs.

Example of correct `dig` output:

ubuntu@ip-10-120-45-213:~$ dig www-staging1.sendhub.com
; <<>> DiG 9.8.1-P1 <<>> www-staging1.sendhub.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 15628
;; flags: qr rd ra; QUERY: 1, ANSWER: 3, AUTHORITY: 0, ADDITIONAL: 0

;; QUESTION SECTION:
;www-staging1.sendhub.com.                IN        A

;; ANSWER SECTION:
www-staging1.sendhub.com.        240        IN        CNAME        sb-lb1.sendhub.com.
sb-lb1.sendhub.com.        240        IN        A        54.227.243.28
sb-lb1.sendhub.com.        240        IN        A        107.20.144.118

;; Query time: 49 msec
;; SERVER: 172.16.0.23#53(172.16.0.23)
;; WHEN: Wed Oct  2 20:48:31 2013
;; MSG SIZE  rcvd: 89

