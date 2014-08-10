cloud-ssh
=========

Wrapper for ssh which enchance work with cloud providers.

In times of digital clouds, servers come and go, and you barely remember its names and addresses. This tiny tool provide fuzzy search (yeah like SublimeText) for your instances list, based on tags, security groups and names. 

Check releases for latest version: https://github.com/buger/cloud-ssh/releases

Here is few examples:

```
sh-3.2$ # Lets say i want connect to server called stage-matching
sh-3.2$ ./cloud-ssh leon@stama
Found config: /Users/buger/.ssh/cloud-ssh.yaml
Found clound instance:
Cloud: granify_ec2      Matched by: aws:autoscaling:groupName=stage-matching    Addr: ec2-50-200-40-200.compute-1.amazonaws.com

Welcome to Ubuntu 12.04 LTS (GNU/Linux 3.2.0-25-virtual x86_64)
```

If there are more then 1 server matching your query, it will ask you to choose one:
```
sh-3.2$ # I want to check one of my CouchBase servers
sh-3.2$ ./cloud-ssh ubuntu@couch
Found config: /Users/buger/.ssh/cloud-ssh.yaml
Found multiple instances:
1)  Cloud: granify_ec2  Matched by: Name=couchbase-02   Addr: ec2-50-200-40-201.compute-1.amazonaws.com
2)  Cloud: granify_ec2  Matched by: Name=couchbase-03   Addr: ec2-50-200-40-202.compute-1.amazonaws.com
3)  Cloud: granify_ec2  Matched by: Name=couchbase-04   Addr: ec2-50-200-40-203.compute-1.amazonaws.com
4)  Cloud: granify_ec2  Matched by: Name=couchbase-01   Addr: ec2-50-200-40-204.compute-1.amazonaws.com
5)  Cloud: granify_ec2  Matched by: Name=couchbase-05   Addr: ec2-50-200-40-205.compute-1.amazonaws.com
Choose instance: 1
Welcome to Ubuntu 12.04.4 LTS (GNU/Linux 3.2.0-58-virtual x86_64)
```

Nice, right? More over, cloud-ssh can act as full ssh replacement, since it just forward all calls to ssh command. 

## Configuration 

By default it checks your environment for AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables. If you want advanced configuration you can create `cloud-ssh.yaml` in one of this directories: ./ (current), ~/.ssh/, /etc/

Note that you can define multiple clouds, per provider, if you have multi-datacenter setup or just different clients. Cloud name will be included into search term, so you can filter by it too!

Right now only 2 data cloud providers supported: Amazon EC2 and DigitalOcean. 

Example configuration:
```
gran_ec2: # cloud name, used when searching
    provider: aws 
    region: us-east-1
    access_key: AAAAAAAAAAAAAAAAA
    secret_key: BBBBBBBBBBBBBBBBBBBBBBBBB
    default_user: ubuntu
gran_digital:
    provider: digital_ocean
    client_id: 111111111111111111
    api_key: 22222222222222222
```


## Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Added some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create new Pull Request
