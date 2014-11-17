# Using the Dockerfile

This Dockerfile will build an image running Sentinel, RedSkull, and
Consul. Once you've run `go build` you can then run `docker build -t
redskull .` to get a Docker image built. 


# Assumptions

It assumes your `docker0` interface is left with the stock setup. If
you've modified it you will need to modify the JOIN IP in the
Dockerfile.

It also assumes an entirely stock Docker network config. On Port 8000 of
the container IP will be Red Skull, and Sentinel will be on the stock
port of 26379.

As RedSkull uses environment variables for config you can pass them in
the `docker run` command to change them if needed. Normally you won't
need to.

