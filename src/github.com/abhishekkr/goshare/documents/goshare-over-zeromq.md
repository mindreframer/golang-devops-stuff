## Using GoShare over HTTP

#### By Word Stream

```ASCII

[DB-Action] [Task-Type] ([Time-Dot]) ([Parent-NameSpace]) {[Key ([Val...])] OR [DB-Data...]}

```

the specifics for these components have been explained under [User Concepts]() before

> * if you are coding in Golang, you can directly utilize ZmqRequest from "[golzmq](http://github.com/abhishekkr/gol/golzmq)" *
> * otherwise, just prepare the word-stream and pass it as bytes in your favorite ZeroMQ Request library *

---

#### Word Stream for major usable scenarios


> simple Key Values

* Push a simple Key Value ``` push default kernel linux ```

* Push multiple Key Value as CSV ``` push default-csv kernel,linux ```

* Push multiple Key Value as JSON ``` push default-json {"kernel":"linux"} ```

* Read a simple key, default response into CSV ``` read default kernel ```

* Read a simple key into JSON response ``` read default-json ["kernel"] ```

* Read multiple keys from JSON into JSON response ``` read default-json ["kernel","os"] ```

* Delete a simple key ``` delete default kernel ```

* Delete multiple keys from CSV ``` delete default-csv kernel,os ```


> Namespaced Key Values

* Push a namespace key ``` push ns software:internet:browser chrome ```

* Push multiple namespace keys from CSV ``` push ns-csv software:internet:browser,chrome\nsoftware:internet:downloader,wget ```

* Push a namespace key with parent-namespace ``` push ns-default-parentNS software internet:browser chrome ```

* Push multiple namespace keys from JSON with parent-namespace ``` push ns-json-parentNS {"software:internet:browser":"chrome","software:internet:downloader":"wget"} ```

* Read all namespace value under given key, default as csv ``` read ns "software:internet" ```

* Read all namespace value under given key, fixed as csv ``` read ns-csv "software:internet" ```

* Read all namespace value under given key, response as JSON ``` read ns-json-parentNS software ["internet"] ```

* Delete all namespace value under given key ``` delete ns software:internet ```


> TimeSeries Key Values

* Push a tsds key val ``` push tsds 2014 5 25 11 53 10 node:mymachine:memfree 4 ```

* Push multiple tsds keys from CSV ``` push tsds-csv 2014 5 25 11 53 10 node:mymachine:memfree,4\nnode:mymachine:diskfree,20 ```

* Push multiple NOW timeseries keys from JSON at TimeDot of getting stored ``` push now-json {"node:mymachine:memfree":"4","node:mymachine:diskfree":"20"} ```

* Push multiple tsds keys from JSON with parent-namespace ``` push tsds-json-parentNS 2014 5 25 11 53 10 node:mymachine {"memfree":"4","diskfree":"20"} ```

* Read all tsds values under given key, response as JSON ``` read tsds-json ["node:mymachine"] ```

* Read all tsds value under given key, response as JSON ``` read tsds-json-parentNS node ["mymachine","othermachine"] ```

* Delete all tsds value under given key ``` delete tsds node ```

---
