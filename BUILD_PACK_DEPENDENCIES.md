When installing dependencies, whether it is through python-pip or something else, sometimes sub-dependencies may be missing.  The tricky part of this kind of problem is that it needs to be addressed at 3 levels:

    - Application container level (so the application can get the dependency and run)
    - Production ShipBuilder environment level (so future applications will have the capability)
    - ShipBuilder installation/initialization level (so future installs do not have the same problem)

Here are the steps to creating a lasting resolution for this kind of problem:

I. Determine what packages and/or steps are required to resolve the issue in the form of a list of commands.
II. Update the container for the affected application and prove the dependency issue is resolved with commands from step 1.
III. Update the base container for the build-pack.
IV. Update the associated build-pack package definition in build-packs/<name>

--

Example:

What: On a python application, trying to test gevent (a different worker within gunicorn). When trying to install, I get the following: 
    remote: 1:44 gcc -pthread -fno-strict-aliasing -DNDEBUG -g -fwrapv -O2 -Wall -Wstrict-prototypes -fPIC -I/usr/include/python2.7 -c gevent/core.c -o build/temp.linux-x86_64-2.7/gevent/core.o
    remote: 1:44 
    remote: 1:44 In file included from gevent/core.c:253:0:
    remote: 1:44 
    remote: 1:44 gevent/libevent.h:9:19: fatal error: event.h: No such file or directory
    remote: 1:44 
    remote: 1:44 compilation terminated.
    remote: 1:44 
    remote: 1:44 error: command 'gcc' failed with exit status 1
    remote: 1:44 
    remote: 1:44 ----------------------------------------
    remote: 1:44 Command /usr/bin/python -c "import setuptools;__file__='/app/src/build/gevent/setup.py';exec(compile(open(__file__).read().replace('\r\n', '\n'), __file__, 'exec'))" install --single-version-externally-managed --record /tmp/pip-douWwW-record/install-record.txt failed with error code 1
    remote: 1:44 Storing complete log in /root/.pip/pip.log
    remote: 1:44 RETURN_CODE: 1
    remote: 1:44 $ sudo lxc-stop -k -n voice-s1
    remote: build failed

I. Determine what packages and/or steps are required to resolve the issue in the form of a list of commands.

Okay, so it looks like we are probably missing a library.  Let's clone the applications container and figure out how to resolve this problem.

    $ ssh sb.example.com
    $ sudo lxc-clone -s -B zfs -o my-app -n my-app-tmp  # Create a temporary snapshot clone of the app.
    $ sudo lxc-start -d -n my-app-tmp                   # Start the container in daemon mode.
    $ sudo lxc-attach -n my-app-tmp /bin/bash           # Attach an interactive shell to the container.
    root@my-app-tmp:~# apt-cache search libevent
    libevent-2.0-5 - Asynchronous event notification library
    libevent-core-2.0-5 - Asynchronous event notification library (core)
    libevent-dbg - Asynchronous event notification library (debug symbols)
    libevent-dev - Asynchronous event notification library (development files)
    libevent-extra-2.0-5 - Asynchronous event notification library (extra)
    libevent-openssl-2.0-5 - Asynchronous event notification library (openssl)
    libevent-pthreads-2.0-5 - Asynchronous event notification library (pthreads)
    libeventviews4 - event view library
    libverto-libevent1 - Event loop abstraction for Libraries - libev
    python-eventlet - concurrent networking library for Python
    event-rpc-perl - dummy package to install libevent-rpc-perl
    libev-dev - static library, header files, and docs for libev
    libev-libevent-dev - libevent event loop compatibility wrapper for libev
    libev-perl - Perl interface to libev, the high performance event loop
    libev4 - high-performance event loop library modelled after libevent
    libevent-1.4-2 - asynchronous event notification library
    libevent-core-1.4-2 - asynchronous event notification library (core)
    libevent-execflow-perl - High level API for event-based execution flow control
    libevent-extra-1.4-2 - asynchronous event notification library (extra)
    libevent-loop-ruby - Transitional package for ruby-event-loop
    libevent-loop-ruby1.8 - Transitional package for ruby-event-loop
    libevent-perl - generic Perl event loop module
    libevent-rpc-perl - Event based transparent Client/Server RPC framework
    libevent1-dev - development libraries, header files and docs for libevent
    libeventdb-dev - library that provides access to gpe-calendar data [development]
    libeventdb2 - library that provides access to gpe-calendar data [runtime]
    libeventdb2-dbg - library that provides access to gpe-calendar data [debugging]
    libeventmachine-ruby - Transitional package for ruby-eventmachine
    libeventmachine-ruby-doc - Transitional package for ruby-eventmachine
    libeventmachine-ruby1.8 - Transitional package for ruby-eventmachine
    liblua5.1-event-dev - libevent development files for Lua version 5.1
    liblua5.1-event0 - asynchronous event notification library for Lua version 5.1
    libpoe-loop-event-perl - POE event loop implementation using Event
    python-gevent - gevent is a coroutine-based Python networking library
    python-gevent-dbg - gevent is a coroutine-based Python networking library - debugging symbols
    python-gevent-doc - gevent is a coroutine-based Python networking library - documentation
    ruby-event-loop - simple signal system and an event loop for Ruby
    ruby-eventmachine - Ruby/EventMachine library
    unworkable - efficient, simple and secure bittorrent client

Let's try installing libevent-dev to see if resolving the issue is that easy.

    root@my-app-tmp:~# sudo apt-get install -y libevent-dev

And try to install gevent again to verify things now work as expected:

    root@my-app-tmp:~# pip install gevent
    Downloading/unpacking gevent
      Downloading gevent-0.13.8.tar.gz (300Kb): 300Kb downloaded
      Running setup.py egg_info for package gevent
    
    Downloading/unpacking greenlet (from gevent)
      Downloading greenlet-0.4.1.zip (75Kb): 75Kb downloaded
      Running setup.py egg_info for package greenlet
    
    Installing collected packages: gevent, greenlet
      Running setup.py install for gevent
        building 'gevent.core' extension
        gcc -pthread -fno-strict-aliasing -DNDEBUG -g -fwrapv -O2 -Wall -Wstrict-prototypes -fPIC -I/usr/include/python2.7 -c gevent/core.c -o build/temp.linux-x86_64-2.7/gevent/core.o
        gcc -pthread -shared -Wl,-O1 -Wl,-Bsymbolic-functions -Wl,-Bsymbolic-functions -Wl,-z,relro build/temp.linux-x86_64-2.7/gevent/core.o -levent -o build/lib.linux-x86_64-2.7/gevent/core.so
        Linking /home/ubuntu/build/gevent/build/lib.linux-x86_64-2.7/gevent/core.so to /home/ubuntu/build/gevent/gevent/core.so
    
      Running setup.py install for greenlet
        gcc -pthread -fno-strict-aliasing -DNDEBUG -g -fwrapv -O2 -Wall -Wstrict-prototypes -fPIC -fno-tree-dominator-opts -I/usr/include/python2.7 -c /tmp/tmpfi3Bi3/simple.c -o /tmp/tmpfi3Bi3/tmp/tmpfi3Bi3/simple.o
        /tmp/tmpfi3Bi3/simple.c:1:6: warning: function declaration isn’t a prototype [-Wstrict-prototypes]
        building 'greenlet' extension
        gcc -pthread -fno-strict-aliasing -DNDEBUG -g -fwrapv -O2 -Wall -Wstrict-prototypes -fPIC -fno-tree-dominator-opts -I/usr/include/python2.7 -c greenlet.c -o build/temp.linux-x86_64-2.7/greenlet.o
        greenlet.c: In function ‘g_switch’:
        greenlet.c:593:5: warning: ‘err’ may be used uninitialized in this function [-Wuninitialized]
        gcc -pthread -shared -Wl,-O1 -Wl,-Bsymbolic-functions -Wl,-Bsymbolic-functions -Wl,-z,relro build/temp.linux-x86_64-2.7/greenlet.o -o build/lib.linux-x86_64-2.7/greenlet.so
        Linking /home/ubuntu/build/greenlet/build/lib.linux-x86_64-2.7/greenlet.so to /home/ubuntu/build/greenlet/greenlet.so
    
    Successfully installed gevent greenlet
    Cleaning up...

And voila! It works.  Okay, so we now know the missing ubuntu package is `libevent-dev`, and the command we need is:

    sudo apt-get install -y libevent-dev

Logout of the temporary container and cleanup after ourselves:

    root@my-app-tmp:~# exit
    exit
    $ sudo lxc-stop -k -n my-app-tmp
    $ sudo lxc-destroy -n my-app-tmp

II. Update the container for the affected application and prove the dependency issue is resolved with commands from step 1.

Fairly straightforward; start the base container for the application and run the command:

    $ sudo lxc-start -d -n my-app
    $ sudo lxc-attach -n my-app -- sudo apt-get install -y libevent-dev
    $ sudo lxc-stop -k -n my-app

III. Update the base container for the build-pack.

    $ sudo lxc-start -d -n base-python
    $ sudo lxc-attach -n base-python -- sudo apt-get install -y libevent-dev
    $ sudo lxc-stop -k -n base-python

IV. Update the associated build-pack package definition in build-packs/<name>

Edit build-packs/python/container-packages and append `libevent-dev` to the list.

Commit your changes and you will be all set (and never have to deal with that particular error again!).

