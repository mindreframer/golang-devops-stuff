Let's be clear: on Heroku, the rollback command does exit immediately or almost immediately, but the actual rollback process takes around 60-120 seconds (in my experience), during which period the service is unavailable/intermittently available/slow.

I suspect what is actually happening with Heroku rollbacks is that the command just pushes the rollback operation onto some kind of queue, and then exits.  Their system then probably has some semi-intelligent logic which is able to makes certain guarantees about always eventually being able to bring the application back online at the previous version.

-

With ShipBuilder, it's true that presently a rollback operation will take as long as a normal deploy.

One way rollbacks could be sped up would be not destroy dynos when they are shut down, and to keep track of the previous set of dyno's on which the app was running.  Then, instead of having to do what is equivalent to a full deploy, you could just turn the old dynos back on and then shutdown the new ones.

HOWEVER, there are caveats to this approach:

1. Significant additional complexity in how old containers (dynos) are managed - right now, we destroy the containers at the same time that we shut them down, which keeps the situation very simple.

2. Tracking of what containers were previously being used must be added to the JSON config file and continuously updated.

3. Some kind of old container management layer will be required to manage expired containers.  If too many are left on a node ( > 200 in my experience), LXC will start taking an exceedingly long time to determine it's state for `lxc-ls --fancy`.  If there are thousands or tens of thousands of unused containers on a node, ShipBuilders status monitor will not be able to check at its typical interval (which something like ~30 seconds).

4. Edge cases: What if one or more of the nodes the app previously ran on is no longer available? The app won't be taken offline, but the rollback also won't work.  I guess if any of the old containers fail to boot, you could fall back to using the current method - a "full redeploy" rollback.

With that being said, if a current deploy takes 3-6 minutes, I would expect that using this approach you could reduce it down to 1-3 minutes, as the container rsync'ing process does consume a fair amount of the total time during a deploy.
