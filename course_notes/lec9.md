# lecture 9: More Replication, CRAQ

*every node can serve read operation but still preserve strong consistency*

## chain replication

### cr

![arch](./figures/lec9-1.png)

### craq

![arch](./figures/lec9-2.png)

![arch](./figures/lec9-3.png)

*tail sees all the read and write* -> total order

*not partition-proof*, need external *configuration manager* (use RAFT, Paxos or Zookeeper).

## pros and cons

1. pros
    + better read performance but still strong consistency which zookeeper cannot provide.
    + raft leader processes all the read/write operation and send rpc to each replicas
    + each cr replicas processes write operation and send only one rpc to successor and only one replica serves read operation
2. cons
    + need to wait for any server in the chain.
    + fault tolerance rely on external service.
    + Raft or Paxos need not to wait for a follower, since it only needs to acquire majority vote. Therefore, easier to deploy to different datacenter. (need not to wait a follower in a distant datacenter)

## Questions

```
Q: What are the tradeoffs of Chain Replication vs Raft or Paxos?

A: Both CRAQ and Raft/Paxos are replicated state machines. They can be
used to replicate any service that can be fit into a state machine
mold (basically, processes a stream of requests one at a time). One
application for Raft/Paxos is object storage -- you'll build object
storage on top of Raft in Lab 3. Similarly, the underlying machinery
of CRAQ could be used for services other than storage, for example to
implement a lock server.

CR and CRAQ are likely to be faster than Raft because the CR head does
less work than the Raft leader: the CR head sends writes to just one
replica, while the Raft leader must send all operations to all
followers. CR has a performance advantage for reads as well, since it
serves them from the tail (not the head), while the Raft leader must
serve all client requests.

However, Raft/Paxos and CR/CRAQ differ significantly in their failure
properties. Raft (and Paxos and ZooKeeper) can continue operating
(with no pauses at all) even if a minority of nodes are crashed, slow,
unreliable, or partitioned. A CRAQ or CR chain must stop if something
like that goes wrong, and wait for a configuration manager to decide
how to proceed. On the other hand the post-failure situation is
significantly simpler in CR/CRAQ; recall Figures 7 and 8 in the Raft
paper.
```
---

```
Q: What alternatives exist to the CRAQ model?

A: People use Chain Replication (though not CRAQ) fairly frequently.

People use quorum systems such as Paxos and Raft very frequently (e.g.
ZooKeeper, Google Chubby and Spanner and Megastore).

There are lots of primary/backup replication schemes that you can view
as similar to a Chain Replication chain with just two nodes, or with the
primary sending to all replicas directly (no chain). GFS is like this.

The main technique that people use to keep strong consistency but allow
replicas to serve reads is leases.
```
