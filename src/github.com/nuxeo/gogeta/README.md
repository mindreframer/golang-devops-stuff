Gogeta
======

Gogeta is a dynamic reverse proxy which configuration is based on [etcd](https://github.com/coreos/etcd). It provides real time dynamic reconfiguration of routes without having to restart the process.

It is part of the nuxeo.io infrastructure.


How it works
------------

It is basically an HTTP cluster router that holds its configuration in etcd. Default behavior
is to use the IoEtcdResolver :

  * client asks for mycustomdomain.com
  * proxy looks at `/nuxeo.io/domains/mycustomdomain.com/[type,value]`
  * if type is io container we look for `/nuxeo.io/envs/{value}/[ip,port]`
  * the request is proxied to `http://{ip}:{port}/`
  * if type is uri
  * the requestion is proxies to the value `/nuxeo.io/domains/mycustomdomain.com/value`

It also provides to custom resolvers :

  * EnvResolver : it serves `http://{envid}.local/ to the host referenced at `/nuxeo.io/envs/{envid}/[ip,port]`
  * DummyResolver : it always proxies to `http://localhost:8080/`


Configuration
-------------

Several parameters allow to configure the way the proxy behave :

 * `domainDir` allows to select the prefix of the key where it watches for domain
 * `envDir` allows to select the prefix of the key where it watches for environments
 * `etcdAddress` specify the address of the `etcd` server
 * `port` port to listen
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
