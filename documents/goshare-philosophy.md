## GoShare Philosophy

##### *(in short)* **GoShare can be used to have *persistent* Time-Series, Namespace-KeyVal or typical Key-Val data-store over HTTP and ZeroMQ communication capability as independent server (or built-in for Golang applications).**

---

#### What the hell is it?
GoShare is an engine for Time-Series Data-Store, a datastore tailor-fit built to suit the requirement of storing varying states of any type of attribute along the timeline.
It's not a database built for generic purpose used for TimeSeries data, but a datastore which will keep on improving to serve only this kind of requirement from core up.

---

#### *Uh*, which solution would even use it?
It's kind of datastore which is perfect for requirements of a real-time monitoring system. Any kind of analytics system which requires change of attribute values along the stretch of a timeline could just plug-in and use it with a made for each other ease.
There could be a lot more usable case for a timeseries datastore, these are the one I mainly ideated it for.

---

#### *In the name of God*, why re-invent the wheel?
There are not a page long list of alternatives for this.

When I ideated it, the only solutions available were not self-dependent (using Zookeeper, MongoDB). Hence they were not actually alternative of an independent (born-for-it) Time-Series Data-Store. Also, I don't consider (stupid) closed-source projects for my FOSS initiatives.
I needed such independent datastore for another monitoring project of mine where I didn't wanted to have more than one services to worry about for the core of my montitoring service. That would have like Watchmen for Watchmen for...
Then later a project started (["InfluxDB"](http://influxdb.org/))  when I already started this. Which is in some way similar to this project but with much larger workforce behind it. It's also OpenSource, written in Golang.

I didn't re-invented the entire wheel just the improved upon the design and made it suit the road. The core of this datastore sits and inter-woven key-val store ["leveldb"](https://code.google.com/p/leveldb/) (it's an awesome, made for performance key-val store mechanism by some (x)Googlers which also the core of super-awesome [Riak](http://basho.com/riak/)).

I also wanted to give it an awesome super communication channel which is not there in other alternative last I checked. Currently it allows communication ZeroMQ Req/Rep connections along with HTTP calls.

For data encoding, it supports key-val, csv and json for now. Msgpack (Protobuf and Cap'n'Proto) shall be in within a month or two (by July/2014).

If you are thinking on "why CSV support". The main reason for that is since it was ideated with monitoring solution in mind, which aim on supporting agents on node supporting plugins for monitoring data. Plug-ins supported for written in any technology, even a bash script doing sed+grep over status files with most easier encoding available as csv. Idea is supporting wide range at features where it will help better & quicker adoption.

Then we've the power of ZeroMQ to communicate with it (as mentioned before) enabling the awesomeness of supercharged socket communication with minimal formality possible.

---

#### It's an Engine, so where is the Train?
The repo itself provides an extra piece of code allowing creation of service/daemon binaries to start using it and trying the power real quick.

As for a proper Train driving this engine, in parallel I've started design of [MomentDB](https://github.com/abhishekkr/momentdb) which will utilize it and enable having multiple style load-balanced, replicated and fail-over mechanisms for it. Just hold on for few months, design is almost done... development spike has started in pieces.

The monitoring system which lead to its creation, has also seen the light of spiking. It can be kept track of at [ChaacMonitoring](https://github.com/ChaacMonitoring), but there is a very dumb structure there till now. It will show you GoShare in action but not of much utlization.

---

#### It's not, but it is...

It's not built for typical key-val or namespace key-val data storage but on the way of its construction. It also allows to use all the available data-encoding (and the cooler efficient binary ones to come) over HTTP and also awesome ZeroMQ for Key-Val and NameSpace Key-Val persistent storage.

---

[Get QuickStart at GoShare](https://github.com/abhishekkr/goshare/wiki/QuickStart)
