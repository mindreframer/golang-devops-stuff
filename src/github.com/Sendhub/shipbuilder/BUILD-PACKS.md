The process for adding compatibility is pretty simple (I'll be adding these instructions to the SB documentation).

Creating a new type of build-pack:

    - Create a directory for your new build-pack inside build-packs/ (see the `python` build-pack for a good example of precisely what is involved):
        * `container-packages` contains a list of apt packages and libraries to install to the build-pack's base container.  These will be available to all apps of the new build-pack type.
        * `container-custom-commands` should contain any special commands required to create the base container for this build-pack (e.g. for play-framework, it downloads and installs the "play-framework" zipfile).
        * `pre-hook` is a shell script which fetches app-dependencies and can perform any required intermediate steps to prepare your app to be run.

Then once you have the ShipBuilder system installed, you'll need a ProcFile in each apps base directory -- just like you do with Heroku (https://devcenter.heroku.com/articles/procfile).  Then just follow these steps for each app and you'll be set:
    1. Create the app (e.g. `./shipbuilder create myApp ror`).
    2. Add a git remote to your git repo which points at your shipbuilder server (e.g. `git remote add ssh://ubuntu@sb.ruhroh.com/git/myApp`).
    3. Add the apps domain name(s) so they'll be recognized by the load-balancer (e.g. `./shipbuilder domains:add myapp.ruhroh.com -amyApp`)*
    4. `git push` your app to shipbuilder (if needed, see SB docs for additional information).

* Note: Naturally the DNS entry for app domains must point to your load-balancer in order for everything to work.
