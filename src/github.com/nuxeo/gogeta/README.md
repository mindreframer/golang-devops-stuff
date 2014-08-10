


Gogeta
======

[![Build Status](https://travis-ci.org/arkenio/gogeta.png?branch=master)](https://travis-ci.org/arkenio/gogeta)

Gogeta is a dynamic reverse proxy which configuration is based on [etcd](https://github.com/coreos/etcd). It provides real time dynamic reconfiguration of routes without having to restart the process.

It is part of the nuxeo.io infrastructure.


How it works
------------

It is basically an HTTP cluster router that holds its configuration in etcd. Default behavior
is to use the IoEtcdResolver. For instance, when running gogeta like this :

        gogeta -etcdAddress="http://172.17.42.1:4001" \
               -domainDir="/domains" \
               -serviceDir="/services" \
               -templateDir="/usr/local/go/src/github.com/nuxeo/gogeta/templates"

Here is the workflow of the request
  * client asks for mycustomdomain.com
  * proxy looks at `/domains/mycustomdomain.com/[type,value]`
  * if `type` is `service` we look for `/services/{value}/1/location` which value is in the form

        {"host":"172.13.4.3","port":42567}

  * the request is proxied to `http://{host}:{port}/`

  * if `type` is uri
  * the request is proxied to the value of `/domains/mycustomdomain.com/value`


It is possible to have several instances of a service by differenciating them with the `serviceIndex`
key part :

    /services/myService/1/location
    /services/myService/2/location

Gogeta will loadbalance the requests on those two instances using a round robin implementation.


Sample configuration
--------------------

To summarize, here are the keys needed to proxy `customdomain.com` to `172.41.4.5:42654`


    /services/myService/location = {"host":"172.41.4.5", "port": 42654}
    /domains/mycustomdomain.com/type = service
    /domains/mycustomdomain.com/value = myService


Service Status
--------------

Optionnaly, services may have a status. This is a directory that is held at `/services/{serviceName}/{serviceIndex}/status`.
It holds three values:

 * `current` :  The current status of the service in [stopped|starting|started|stopping]
 * `expected`: The expected status of the service [stopped|started]
 * `alive`: a heartbeat that the service must update.

Based on those values, Gogeta will serve wait pages with the according HTTP status code.

Configuration
-------------

Several parameters allow to configure the way the proxy behave :

 * `domainDir` allows to select the prefix of the key where it watches for domain
 * `serviceDir` allows to select the prefix of the key where it watches for environments
 * `etcdAddress` specify the address of the `etcd` server
 * `port` port to listen
 * `templateDir` a template directory for eroor status page
 * `resolverType` : choose the resolver to use
    * `Env` : EnvResolver
    * `Dummy` : DummyResolver
    * by default : IoEtcd

Report & Contribute
-------------------

We are glad to welcome new developers on this initiative, and even simple usage feedback is great.
- Ask your questions on [Nuxeo Answers](http://answers.nuxeo.com)
- Report issues on this github repository (see [issues link](http://github.com/nuxeo/gogeta/issues) on the right)
- Contribute: Send pull requests!


About Nuxeo
-----------

Nuxeo provides a modular, extensible Java-based
[open source software platform for enterprise content management](http://www.nuxeo.com/en/products/ep),
and packaged applications for [document management](http://www.nuxeo.com/en/products/document-management),
[digital asset management](http://www.nuxeo.com/en/products/dam) and
[case management](http://www.nuxeo.com/en/products/case-management).

Designed by developers for developers, the Nuxeo platform offers a modern
architecture, a powerful plug-in model and extensive packaging
capabilities for building content applications.

More information on: <http://www.nuxeo.com/>
