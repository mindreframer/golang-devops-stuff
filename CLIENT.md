Client Commands
---------------

Build the client by running `./build.sh`.

Note:

* Any command that takes an [application-name] either gets the application name from the current directory or it must be specified with `-a<application name>`.


System-wide commands
--------------------

__apps:list__

    apps[:list?]

Lists all applications.


__lb:add__

    lb:add [address]..

Add one or more new load balancers to the system. Updates the load balancer config.


__lb:list__

    lb[:list?]

List all the load balancers.


__lb:remove__

    lb:remove [address]..

Remove one or more load balancers from the system. Updates the load balancer config.


__nodes:add__

    nodes:add [address]..

Add one or more nodes to the system (a node hosts the containers running the actual apps).


__nodes:add__

    nodes:add [address]..

Add one or more nodes to the system (a node hosts the containers running the actual apps).


__nodes:list__

    nodes[:list?]

List all the nodes.


__nodes:remove__

    nodes:remove [address]..

Remove one or more nodes from the system.


Application-specific commands
-----------------------------
__apps:create__

    [apps:]create [application-name] [buildpack]

Alternative flag combinations:
    [apps:]create -a[application-name] [buildpack]

Create an appication named `name` with the build pack `buildpack`. Available buildpacks are:

* python
* playframework2


__apps:clone__

    [apps:]clone [old-application-name] [new-application-name]

Alternative flag combinations:
    [apps:]clone -o[old-application-name] -n[new-application-name]
    [apps:]clone --oldName=[old-application-name] --newName=[new-application-name]

Clone (copy) an application with it's config and processes settings into a new app.


__apps:destroy__

    [apps:]destroy -a[application-name]

Destroy the app with the name `name`.


__config:list__

    config[:list] -a[application-name]

Show all the configuration entries for an application.


__config:get__

    config:get [application-name] variable-name

Return the configuration entry for an application and variable name.


__config:set__

    config:set [variable-name]=[variable-value].. -a[application-name]

Set one or more configuration environment variables for the named application. Redeploys the app.

There is also a `--deferrable=1`/`-d1` flag which can be passed to cause the config change to take effect the next time the app is deployed (avoids the default immediate redeploy).


__config:remove__

    config:remove [variable-name].. -a[application-name]

Delete one or more configuration environment variables for the named application. Redeploys the app.

There is also a `--deferrable=1`/`-d1` flag which can be passed to cause the config change to take effect the next time the app is deployed (avoids the default immediate redeploy).


__deploy__

    deploy revision -a[application-name]

Deploy an application at the given revision (the revision must be available in the local git repository).


__domains:add__

    domains:add [domain-name].. -a[application-name]

Add one or more domains to an application. Updates and reloads the load-balancer immediately; Does NOT redeploy the app.


__domains:list__

    domains:list -a[application-name]

List the domains for an application.


__domains:remove__

    domains:remove [domain-name].. -a[application-name]

Remove one or more domains from an application. Redeploys the app.


__logs__

    logs -a[application-name]

Display the logs for an application. *Not Implemented*


__maintenance:off__

    maintenance:off -a[application-name]

Turns off maintenance mode for an application.


__maintenance:on__

    maintenance:on -a[application-name]

Turns on maintenance mode for an application.


__maintenance:status__

    maintenance[:status?] -a[application-name]

Gets the current maintenance status for an application.  Status values are "on" or "off".


__maintenance:url__

    maintenance:url [url?] -a[application-name]

If `url` is empty, the current maintenance page URL is shown.
If `url` is not empty, will sets the environment variable `MAINTENANCE_PAGE_URL`, which will be used when maintenance-mode is "on".  No redeploy required.
Alternatively, you can also use config:set to a similar effect, with the addition of a full redeploy, e.g.:

    sb config:set MAINTENANCE_PAGE_URL='http://example.com/foo/bar.html' -aMyApp


__pre-receive__

    pre-receive directory old-revision new-revision reference

Internal command automatically invoked by the git repo on pre-receive.


__post-receive__

    post-receive directory old-revision new-revision reference

Internal command automatically invoked by the git repo on post-receive.


__ps:list__

    ps[:list?] -a[application-name]

List the goal and actual running instances of an application.


__ps:scale__

    ps:scale [process-type]=#num#.. -a[application-name]

Update the number of instances for one or more process types. Redeploys the app.


__releases:info__

    releases:info [version] -a[application-name]

Get the release information for an application at the given version. *Not yet implemented*


__releases:list__

    releases[:list?] -a[application-name]

List the most recent 15 releases for an application.


__reset__

    reset -a[application-name]

Reset an the base container for an applications. This will force all dependencies to be freshly downloaded and built during the next deploy.


__rollback__

    rollback [version] -a[application-name]

Rollback an application to a specific version. Note: Version is not optional.


__run__

    run [shell-command?] -a[application-name]

Starts up a temporary container and hooks the current connection to a shell.  If `shell-command` is omitted, by default a bash shell will launched.


Project Compilation
-------------------

Requirements:

- go-alng v1.1
- daemontools (the package which contains `envdir`)

First set up your env:

    echo 'sb.sendhub.com' > env/SB_SSH_HOST
    echo 'admin:password' > env/SB_HAPROXY_CREDENTIALS
    echo 'true' > env/SB_HAPROXY_STATS
    echo "$HOME/.ssh/id_rsa" > env/SB_SSH_KEY

Build the client:

    ./build.sh

Deploy to SB_SSH_HOST:

    ./deploy.sh


Setting a maintenance page URL
--------------------------------

Set your own custom maintenance page URL to be displayed while the app is in maintenance mode.

    sb config:set MAINTENANCE_PAGE_URL='http://example.com/foo/bar.html' -aMyApp


Setting deploy-hooks URL
------------------------

Set a deploy-hook URL to enable things like HipChat room notifications.

    sb config:set DEPLOYHOOKS_HTTP_URL='https://api.hipchat.com/v1/rooms/message?auth_token=<THE_TOKEN>&room_id=<THE_ROOM>' -aMyApp


ShipBuilder Client Configuration Overrides
------------------------------------------

temporary `env` config overrides are possible, just prefix the variable=value before invoking the client:

    $ SB_SSH_HOST=sb-staging.sendhub.com ./shipbuilder config -aMyApp
    info: environmental override detected for SB_SSH_HOST: sb-staging.sendhub.com
    ..
