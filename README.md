```
                                                 ,-.
                                                  ) \
                                              .--'   |
                                             /       /
                                             |_______|
                                            (  O   O  )
                                             {'-(_)-'}
                                           .-{   ^   }-.
                                          /   '.___.'   \
                                         /  |    o    |  \
                                         |__|    o    |__|
                                         (((\_________/)))
                                             \___|___/
                                        jgs.--' | | '--.
                                           \__._| |_.__/
```

Warden in Go, because why not.

* [![Build Status](https://travis-ci.org/vito/garden.png?branch=master)](https://travis-ci.org/vito/garden)
* [![Coverage Status](https://coveralls.io/repos/vito/garden/badge.png?branch=HEAD)](https://coveralls.io/r/vito/garden?branch=HEAD)
* [Tracker](https://www.pivotaltracker.com/s/projects/962374)
* [Warden](https://github.com/cloudfoundry/warden)

# Running

For development, you can just spin up the Vagrant VM and run the server
locally, pointing at its host:

```bash
vagrant up
ssh-copy-id vagrant@192.168.50.4
ssh vagrant@192.168.50.4 sudo cp -r .ssh/ /root/.ssh/
./bin/run-garden-remote-linux
```

This runs the server locally and configures the Linux backend to do everything
over SSH to the Vagrant box.
