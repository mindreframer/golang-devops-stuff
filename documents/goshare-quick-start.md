## GoShare QuickStart

#### Get the freakin' code

* have git client, awesome it's easy ``` git clone https://github.com/abhishekkr/goshare ```

* have svn client, it's fine ``` svn co https://github.com/abhishekkr/goshare ```

* ahhh... have internet, download and uncompress ``` https://github.com/abhishekkr/goshare/archive/v0.8.4.tar.gz ```

---

#### Build your binaries

Make cloned/checkout/downloaded 'goshare' code directory as your current directory.

* have bash/zsh shell, cool ``` ./go-tasks.sh bin ```

* have golang at least
>
> * fetch all dependencies mentioned in 'go-get-pkg.txt'
> * run at cli ``` mkdir ./bin ; cd ./bin ; go build ../zxtra/goshare_service.go ```
>

---

#### Start it up


* if you have a '/tmp' location with write access for current user
> * run ``` ./bin/goshare_server ```

* otherwise
> * run ``` ./bin/goshare_server -dbpath=$ANY_LOCATION_FORDB_CREATION ```

>
> * HTTP listener at 0.0.0.0:9999
> * ZeroMQ Request/Reply listener at 9898, 9797
>
> when started in non-daemon mode, these details get printed on command line as well
>

Now you can just run ``` go run zxtra/gohttp_client.go ``` to fire some read, push and delete calls to GoShare. Or make some of your own.

>
> You'll notice in a particular requirement -dbpath flag has been used to provide location of DB to be passed to GoShare. There are similar other flags available, which are:
>
> * dbpath     : path to create database at; Default: "/tmp/GO.DB"
> * http-uri   : IP to connect while opening http port, Default: "0.0.0.0"
> * http-port  : Port to open http server connection for GoShare, Default: "9999"
> * req-port   : First port to bind ZeroMQ Request/Reply socket server, Default: "9797"
> * rep-port   : Second port to bind ZeroMQ Request/Reply socket server, Default: "9898"
> * cpuprofile : file to log cpu profile data
>
> * config     : json config file to provide all other flag value to override
>

>
> The same set of configuration changes (or a portion of it) can be applied by a configuration file in JSON format, written like
> ```JSON
> {"http-port": "8080"}
> ```
>

So, here configuration applied by JSON file will override configuration applied by flags. If flags are not provided then the default value for them will be assigned to them.

---

#### Ceate, Read, Delete

These actions have been covered in detail at:
* Create : [wiki]*to-be-written*
* Read   : [wiki]*to-be-written*
* Delete : [wiki]*to-be-written*

For this QuickStart section, let's try (assumption you have curl, else visit same URLs in your fav Browser you spoilt kid):

```SHELL
#
curl http://127.0.0.1:9999/get?type=default&key=name

#
curl http://127.0.0.1:9999/put?type=default&key=name&val=ledzep

#
curl http://127.0.0.1:9999/get?type=default&key=name

#
curl http://127.0.0.1:9999/put?type=ns&key=name:full&val=LedZep
curl http://127.0.0.1:9999/put?type=ns-json&dbdata={\"name:first\":\"Led\",\"name:last\":\"Zep\"}

#
curl http://127.0.0.1:9999/get?type=ns&key=name

#
curl http://127.0.0.1:9999/del?type=ns&key=name

#
curl http://127.0.0.1:9999/get?type=ns&key=name

#
curl http://127.0.0.1:9999/put?type=now-json&dbdata={\"node01:webservice:state\":\"up\",\"node01:memfree\":\"4256783\"}

#
curl http://127.0.0.1:9999/get?type=ns&key=node01:webservice
curl http://127.0.0.1:9999/get?type=ns&key=node01

#
curl http://127.0.0.1:9999/del?type=ns&key=node01

#
curl http://127.0.0.1:9999/get?type=ns&key=node01
```

GoShare's HTTP link also allows using POST/PUT for Create and DELETE for Delete tasks as method rather than route based approach. So, to each their own satisfaction.

---
