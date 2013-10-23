ShipBuilder Server Installation
===============================

Requirements
------------
* ShipBuilder Server is compatible with Ubuntu version 12.04 and 13.04; Both have been tested and verified working (as of June 2013)
* Passwordless SSH and sudo access from your machine to all servers involved
* daemontools installed on your local machine
* go-lang v1.1 installed on your local machine
* AWS S3 auth keys - Used to store backups of application configurations and releases on S3 for easy rollback and restoration.

Server Modules
--------------

ShipBuilder is composed of 3 distinct pieces:

* ShipBuilder Server
* Container Node(s) (hosts which run the actual app containers)
* HAProxy Load-Balancer

System Layout and Topology
--------------------------

ShipBuilder can be built out with any layout you want.

Examples

Each module running on separate hosts (3+ machines):

- one machine for ShipBuilder Server
- one or more machines configured as Container Nodes
- one machine as the Load-Balancer

All modules running on a single host (1 machine):

- single machine configured with SB Sever, added as a Node and Load-Balancer

Installation
------------
1. Spin up or allocate the host(s) to be used, taking note of the /dev/<DEVICE> to use for BTRFS/ZFS storage devices on the shipbuilder server and container node(s)

1.b ensure you can SSH without a password, here is an example command to add your public key to the remote servers authorized keys:
```
    ssh -i ~/.ssh/somekey.pem ubuntu@sb.example.com "echo '$(cat ~/.ssh/id_rsa.pub)' >> ~/.ssh/authorized_keys && chmod 600 .ssh/authorized_keys"
```

2. Checkout and configure ShipBuilder (via the env/ directory)
```
    git clone https://github.com/Sendhub/shipbuilder.git
    cd shipbuilder
    cp -r env.example env

    # Set the shipbuilder server host:        
    echo ubuntu@sb.example.com > env/SB_SSH_HOST

    # Set your AWS credentials:
    echo 'MY_AWS_KEY' > env/SB_AWS_KEY
    echo 'MY_AWS_SECRET' > env/SB_AWS_SECRET
    echo 'MY_S3_BUCKET_NAME' > env/SB_S3_BUCKET
```

3. Run Installers:
```
    # For shipbuilder server (make sure this device is a persistent volume as this will be the source of truth):
    ./install/shipbuilder.sh -d /dev/xvdb install

    # For nodes:
    # (note: not necessary to run this if the nodes and server are running on the same machine)
    ./install/node.sh -H ubuntu@node.example.com -d /dev/xvdb install

    # For load-balancer(s):
    ./install/load-balancer.sh -H ubuntu@lb.example.com install
```

4. Compile ShipBuilder locally:
```
    ./build.sh
```

5. Add the load-balancer:
```
    ./shipbuilder lb:add HOST_OR_IP
```

6. Add the node(s):
```
    ./shipbuilder nodes:add HOST_OR_IP1 HOST_OR_IP2..
```

7. Start creating apps!


Port Mappings
=============

Specific ports must be open for each module.

ShipBuilder Server
------------------

- `tcp/22` - Remote SSH access from SB clients (that's you!)
- `tcp/9998` - App logging
- `udp/9998` - HAProxy request logging

Container Node(s)
-----------------

- `tcp/22` - Remote SSH access from SB server
- `tcp/10000-12000` - Must be reachable from load-balancer

HAProxy Load-Balancer
---------------------

- `tcp/22` - Remote SSH access from SB server
- `tcp/80` - HTTP
- `tcp/443` - HTTPS


Health Checks
=============

All web servers must return a 200 HTTP status code response for GET requests to '/', otherwise the load-balancer will think the app is unavailable.




