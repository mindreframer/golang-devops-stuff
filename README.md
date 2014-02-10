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

* [![Build Status](https://travis-ci.org/pivotal-cf-experimental/garden.png?branch=master)](https://travis-ci.org/pivotal-cf-experimental/garden)
* [![Coverage Status](https://coveralls.io/repos/pivotal-cf-experimental/garden/badge.png?branch=HEAD)](https://coveralls.io/r/pivotal-cf-experimental/garden?branch=HEAD)
* [Tracker](https://www.pivotaltracker.com/s/projects/962374)
* [Warden](https://github.com/cloudfoundry/warden)

# Running

For development, you can just spin up the Vagrant VM and run the server
locally, pointing at its host:

```bash
# if you need it:
vagrant plugin install vagrant-omnibus

# then:
librarian-chef install
vagrant up
ssh-copy-id vagrant@192.168.50.5
ssh vagrant@192.168.50.5 sudo cp -r .ssh/ /root/.ssh/
./bin/add-route
./bin/run-garden-remote-linux


# or run from inside the vm:
vagrant ssh
sudo su -
goto garden
./bin/run-garden-linux
```

This runs the server locally and configures the Linux backend to do everything
over SSH to the Vagrant box.
