# lec6 Fault Tolerance: Raft(1)

> Single entity make critical decision. Simple but single point of failure.

+ Mapreduce controlled by one master
+ GFS relies a single master to choose primary for each replication chunk
+ FT primary + backup => relies a single test-and-set server

goal: *Avoid split brain*

```go
replicated test-and-set server: partition

C1 ----- |- S1
         x
C2 ----- |- S2
```

1. client must talk to both server -> *no fault tolerance*
    + worse than have only one server
2. client talk to those it can talk to and regard the other as dead -> split brain
    + maintain network never fails
    + 人工维护

+ *solution*: majority vote.
    + 2f + 1 -> can withstand f failures
    + *Paxos*
    + *VSR*
+ raft library

## software view: raft library

![raft software view](./figures/lec6-1.png)

+ state: the KV table
+ leader use log to sign and *order* the operations
+ leader needs to remember all the committed logs

### time graph

![raft time graph](./figures/lec6-2.png)

+ c1 -> Get(K)
+ after leader knows the command is committed -> Reply(V)

> When restart, no server knows how far they had been have executed before the crash. Later, the leader will send the heartbeat and figure out the commit point.

### software stack interface

![raft interface](./figures/lec6-3.png)

+ client -> server: start(COMMAND)
    + (index, term)
+ Raft notify application layer -> message on a go channel
    + send ApplyMsg to applyCh
    + (command, index)

## 1. Leader Election

+ Q: Why do we need a leader?
    + A: In common no failure case, a leader is more efficient.

1. Election Timer -> Start Election
    + TERM++, REQUEST VOTE
    + After wins, send Append Entry
        + there is a leader
        + reset the election timer
2. randomize election timer 
    + MIN: *broadcastTime* << election timer
        + the average time it takes a server to send RPCs to every server and receive their responses
    + MAX: election timer << *MTBF*
        + the average time between failures for a single server
            1. effect recovery time, depends on failure frequency
            2. the gap between first timer going off the the second timer going off should be longer for assembling votes in case of split vote
    + every time when reset timer, randomly choose the timeout

+ a client send vote request, the old leader will not cause problem
    1. leader in a partition that doesn't have the majority
        + never commit the operation
    2. before leader fails, it send the appendRPC to a subset of followers

> sensible output: same committed sequence
