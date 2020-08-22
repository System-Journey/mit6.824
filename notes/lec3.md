# GFS

## distributed system abstraction

*Big Storage*

### why hard?

+ performance -> sharding
+ faults -> tolerance
+ tolerance -> replication
+ replication -> inconsistency
+ consistency -> low performance

*strong consistency*: single server model.

## GFS

### goal

+ big & fast
+ global/general use
+ sharding
+ automatic recovery

### context

+ Single data center per deployment
+ Internal use: any client can read an file
+ Big sequential access

> 论文被接受的原因并非是分布式、sharding或者容错。而在于huge scale，工业界案例，弱一致性的成功以及单个master的成功。

### Master Data

+ *table1*: file name -> array of chunk handle (nv)
+ *table2*: chunk handle -> list of chunkservers (v)
    + version # (nv)
    + primary (v)
    + lease expiration (v)
+ on disk:
    + Log：可以append，效率比b-tree或者数据库等效率更高，容易batch。
    + checkpoint

#### READ

1. name, offset -> M
2. M find chunk handle for that offset
3. M sends (H, list of s)
    + s with **latest version**
4. c caches handle + chunkserver list
5. c <-> closest chunk server

#### WRITES(record append)

##### No primary or lease expired - on master

+ Find up-to-date replicas => 因此master需要记录version #
    + 不能通过获取chunk server版本号的最大值，可能存在具有最大值版本号的chunk server宕机的情况
+ Pick Primary, Secondary
+ Increments the V #
+ M Tells P, S the V #
    + M -> P: LEASE
+ M wirtes V # to disk

> 若primary无回应，需要等lease过期才能选择新的primary，否则可能出现两个primary：network partition can cause *split brain*.租约的同步也是个问题。

##### With primary

+ c sends data to all (just temporary...), waits
+ c tells P to append
+ p checks that lease hasn't expired, and chunk has space
+ P picks offset *at end of chunk*
+ all replicas told to write at off
+ if all "yes", "success" -> c
    + else "error" -> c
    + 某些chunk server成功写入，某些失败=>不一致。
+ c retries if error

+ *gurantee*
    + 如果append成功则能保证任何读者之后的读能够在文件中发现append的内容
+ *can not gurantee*
    + 失败的读不存在
    + 不同客户端看到的文件内容相同
    + record顺序相同

+ 强一致性：2PC
+ chunk server检测重复append
+ 如果primary宕机，新上位的secondary需要resynchronize.

### Summary

+ good ideas
    + global cluster file system as *universal* infrastructure
    + separation of naming from storage
    + sharding for parallel throughput
    + huge files/chunks to reduce overheads
    + primary to sequence writes => 2PC
    + leases to prevent split-brain chunkserver primaries
+ limitations
    + single master
    + inconsistent append
    + too long for master recovery for some applications
