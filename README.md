Octopus
=======

A network simulator built on UNIX sockets.


Usage
-----

Octopus has usage documentation which can be retrieved by typing

    octopus -h

If you're attempting to test `sqlcluster`, however, you probably want to use the
`test/harness` distributed with the `sqlcluster` source: it will provide options
to Octopus that closely resemble the CTF testing environment.

One of the characteristic features of distributed systems is their
nondeterminism, and Octopus cannot produce completely deterministic network
simulations. That being said, Octopus will make an effort to provide similar
results when run twice with the same seed value and other parameters.


Design
------

Octopus attempts to simulate a variable-latency and lossy fully-connected
network topology between N hosts. It will never violate TCP-like behavior: all
bytes that arrive at the destination are guaranteed to arrive in the order they
were sent, without any data corruption along the way. That being said, Octopus
is free to split up the byte stream in whatever way it desires, to arbitrarily
delay the stream, or to close the stream at any point (discarding any traffic
that was "in-flight" at the time).

Octopus has two primary design goals:

1. To simulate both connection-level and network-level events. A simple network
   simulator might simulate each network link individually, with every
   connection failing independently. However, this poorly models real network
   events, where failures are highly correlated. For instance, a network split
   will cause some of the nodes to be able to communicate amongst themselves,
   but not with the nodes on the other side of the split.

2. To provide roughly-reproducible results. In particular, the system should be
   architected in such a way that running it twice with the same seed on agents
   with roughly the same communication pattern will exercise similar code paths
   and trigger similar application-level bugs.

To satisfy these goals, Octopus has two primary object types: a single network
director, and several point-to-point connections.

We model the network as a completely connected graph between N nodes, where each
edge represents a connection. Each connection has an associated delay and queue
size, which represents the latency of the connection between the nodes and the
(rough) number of bytes that are allowed to be in-flight at any point in time
respectively. Furthermore, each connection has a flag that represents whether it
is currently connected. The network as a whole, then, can be described as a
state machine over the states of each of its N(N-1)/2 connections.

The network director, then, is simply a process which randomly selects a
sequence of state transitions and times at which they occur. These network
events can be point mutations (changing the latency of a single link, for
instance) or bulk operations (disrupting the network along some split). And
since the director's state transitions represent the bulk of the nondeterminism
in the system, seeding its random number generator is sufficient to produce
roughly reproducible network traces.

One side effect of modeling node-to-node connection state as opposed to the
state of individual transport-level connections is that if node A makes a lot of
connections to node B, they will all exhibit similar latency, and all fail at
roughly the same time. It's unclear if this it at all realistic, but the
reproducibility benefits we get from node-to-node state probably outweighs the
cost of the unrealism.


Monkeys
-------

The network director controls several monkeys, whose job it is to wreak havok
across the network in a particular way:

- The **Latency Monkey** selects a single network link and manipulates its
  latency.
- The **Jitter Monkey** selects a single network link and maniuplates its
  latency jitter (Octopus uses "jitter" to refer to random pertubations added to
  the base latency. A connection with higher jitter will have more variance in
  the latency of individual chunks of data).
- The **Lag Split Monkey** partitions the agents into two groups and makes
  connections between agents in different partitions significantly slower for
  some amount of time.
- The **Link Monkey** selects a single network link and terminates it (dropping
  any in-flight data). It also prohibits new connections for some period of
  time.
- The **Net Split Monkey** partitions the agents into two groups and terminates
  any network links between agents in different halves of the partition. It also
  prohibits new connections for some period of time.
- The **Freeze Monkey** freezes an agent in place (similar to pressing `Ctrl-Z`
  in a terminal) for some period of time.
- The **Murder Monkey** kills an agent's process and respawns it after some
  period of time.
- The **SPOF Monkey** tries to detect if a node is a single point of failure by
  netsplitting it away from the rest of the cluster. It only restores network
  access to that node if the remainder of the cluster can make progress without
  it.

Each monkey enters the fray after some set delay, and will act in a Poisson
fashion until the Octopus run terminates.

Each monkey is also controlled by an ambient "intensity," which is a number
that oscillates sinusoidally between 0 and 1, starting at 0, with a period of 30
seconds. Each monkey treats intensity differently, but in most cases they will
act less severely (or not at all) when intensity is low, and will unleash their
full fury when intensity is highest.
