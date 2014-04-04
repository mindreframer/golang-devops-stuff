OSTENT
======

[**View Demo**](http://demo.ostrost.com/)

![screenshot](https://github.com/rzab/ostent/raw/master/screenshot.png)

   - Memory usage
   - Network traffic
   - Disks usage
   - CPU load
   - Processes
   - to be continued

Everything is on real-time display only, 1 second refresh.
A hosted service with graphs, history, aggregation etc.,
to leave the machines out of it, is bound to happen.
ostent is inteded to be an agent of sort,
but however it goes it's a stand-alone app
and any service connection will be opt-in and optional.

Download
--------

   - [Linux 64bits](https://OSTROST.COM/ostent/releases/latest/Linux x86_64/ostent)
   - [Linux 32bits](https://OSTROST.COM/ostent/releases/latest/Linux i686/ostent)
   - [Darwin](https://OSTROST.COM/ostent/releases/latest/Darwin x86_64/ostent)
   - _Expect \*BSD builds surely_

A single executable without dependecies, has no config, makes no files of it's own.
Self-updates: new releases will be deployed automatically, sans page reload yet.

Laziest install: `curl -sSL https://github.com/rzab/ostent/raw/master/lazyinstall.sh | sh -e`

`ostent` accepts optional `-b[ind]` argument to set specific IP and/or port to bind to, otherwise any machine IP and port 8050 by default.

   - `ostent -bind 127.1` # [http://127.0.0.1:8050/](http://127.0.0.1:8050/)
   - `ostent -bind 192.168.1.10:8051` # port 8051
   - `ostent -bind 8052` # any IP, port 8052

Feedback & contribute
---------------------

[Please do](https://github.com/rzab/ostent/issues/new). Ideas, bugs, pull requests, anything.

Running the code
----------------

1. **`git clone https://github.com/rzab/ostent.git`**

2. **`cd ostent`** `# the project directory`

3. **`export GOPATH=$GOPATH:$PWD`** `# the current directory into $GOPATH`

4. **`go get github.com/jteeuwen/go-bindata/go-bindata`**

5. **`scons`** to generate required `src/ostential/{assets,view}/bindata.devel.go`. It's either scons, or run **manually**:
   ```sh
      go-bindata -pkg view   -o src/ostential/view/bindata.devel.go   -tags '!production' -debug -prefix templates.min templates.min/...
      go-bindata -pkg assets -o src/ostential/assets/bindata.devel.go -tags '!production' -debug -prefix assets        assets/...
   ```

6. Using [rerun](https://github.com/skelterjohn/rerun), it'll go get the remaining dependecies:

	**`go get github.com/skelterjohn/rerun`**

7. **`rerun ostent`**

Go packages
-----------

`[src/]ostential` is the core package.

`[src/]ostent` is the main (as in [Go Program execution](http://golang.org/ref/spec#Program_execution)) package:
rerun will find `main.devel.go` file; the other `main.production.go` (used when building with `-tags production`)
is the init code for the distributed [binaries](#download): also includes
[goagain](https://github.com/rcrowley/goagain) recovering and self-updating via [go-update](https://github.com/inconshreveable/go-update).

Templates
---------

HTML templates in this repository are actually **generated** outside.
I'm OK with publishing the source templates, but the generation depends on
[amber](https://github.com/eknkc/amber),
[react-tools](https://www.npmjs.org/package/react-tools) with Node.js and
another set of scons rules.
The generation makes the HTML templates and propagates the layout into
[React.js](http://facebook.github.io/react/) objects (`assets/js/gen/build.js`).
It's just not that straight-forward.

So for now, the repo has `assets/js/gen/build.js`, `templates.min/` -- the _actual_ templates
and _somewhat readable_ `templates/` for reference.

The assets
----------

The [binaries](#download), to be stand-alone, have the assets (including `templates.min/`) embeded.
Unless you specifically `go build` with `-tags production`, they are not embeded for the ease of development:
with `rerun ostent`, asset requests are served from the actual files.
