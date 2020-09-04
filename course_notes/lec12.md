# lec12: Distributed Transactions

*concurrency control* + *atomic commit*

## motivation

+ data **sharded on many servers**, lots of clients
+ client application actions involve multiple reads and writes
+ *goal:* hide interleaving and failure from clients

## traditional plan: transactions

```
example transactions:
  x and y are bank balances -- records in database tables
  x and y are on different servers (maybe at different banks)
  x and y start out as $10
  T1 and T2 are transactions
    T1: transfer $1 from x to y
    T2: audit, to check that no money is lost
  T1:             T2:
  begin_xaction   begin_xaction
    add(x, 1)       tmp1 = get(x)
    add(y, -1)      tmp2 = get(y)
  end_xaction       print tmp1, tmp2
                  end_xaction
```

### correct behavior for a transaction

*ACID*:

1. Atomic：尽管存在故障的情况，事务的所有操作要么全都执行，要么全都不执行
2. Consistent：遵循应用程序明确的不变式
3. Isolated：事务之间不相互干扰--*serializable*
4. Durable：已提交的事务是持久化的

*problem*:

ACID for distributed transactions?

#### serializable

+ 多个并发事务的最终结果包含了DB的输出和数据的改变
+ 多个并发事务的最终结果是serializable的，如果：
    1. 存在一个事务的串行执行顺序能够产生和实际执行相同的结果

## overview

+ *concurrency control* (to provide isolation/serializability)
+ *atomic commit*(to provide atomicity despite failure)

---

## concurrency control

### classification

1. pessimistic
    + lock records before use
    + conflicts cause delays
2. optimistic
    + use records without locking
    + commit checks if reads/writes were serializable
    + conflict causes abort+retry
    + called Optimistic Concurrency Control (OCC)

#### discussion

1. 冲突频繁时，悲观锁更快
2. 冲突很少发生时，乐观锁更快

### "Tow-phase locking" to implement serializability

#### *2PL definition:*

+ a transaction must acquire a record's lock before using it
+ a transaction must hold its locks until *after* commit or abort

#### *details:*

+ each database record has a lock, if distributed, the lock is typically stored at the record's server
    + an executing transaction acquires locks as needed, at the first use
    + all locks are exclusive
+ *strong strict two-phase locking*

#### hold until after commit/abort

1. not a serializable execution: 释放之后
    + T2 releases x's lock after get(x)
    + T1 could then execute between T2's get()s
    + T2 would print 10,9
2. cascading abort
    + T1 writes x, then releases x's lock
    + T2 reads x and prints
    + T1 then aborts

#### may cause deadlock and forbid a correct execution

+ the system must detect and abort a transaction

---

# distributed transactions versus failures

```
example:

x and y are on different "worker" servers
  suppose x's server adds 1, but y's crashes before subtracting?
  or x's server adds 1, but y's realizes the account doesn't exist?
  or x and y both can do their part, but aren't sure if the other will?
```

+ goal: atomic commit

## two-phase commit: multi-server transactions

### 2PC without failures

+ TC sends put(), get(), &c RPCs to A, B
  + The modifications are tentative, only to be installed if commit.
+ TC gets to the end of the transaction.
+ TC sends PREPARE messages to A and B.
+ If A is willing to commit,
  + A responds YES.
  + then A is in "prepared" state.
+ otherwise, A responds NO.
+ Same for B.
+ If both A and B say YES, TC sends COMMIT messages to A and B.
+ If either A or B says NO, TC sends ABORT messages.
+ A/B commit if they get a COMMIT message from the TC.
  + I.e. they write tentative records to the real DB.
  + And release *the transaction's locks* on their records.
+ A/B acknowledge COMMIT message.

### some questions

1. What if B crashes and restarts?
  + B sent YES before crash, B must remember decision including modified data
  + if B reboot, and disk says YES but no commit
    + ask TC or wait for TC to re-send
  + B must continue to hold the transaction's locks
  + after receiving COMMIT, B can copy modified apply changes to real data
2. What if TC crashes and restarts?
  + TC might have sent COMMIT before crash, must remember
    + since one worker may already have committed
  + repeat COMMIT if it crashes and reboots
    + or if a participant asks
  + participant must filter out duplicate COMMITs (using TID)
3. What if B times out or crashes while waiting for PREPARE from TC?
  + B has not yet responded to PREPARE, so TC can't have decided commit so B can unilaterally abort, and release locks respond NO to future PREPARE
4. What if B replied YES to PREPARE, but doesn't receive COMMIT or ABORT?
  + B cannot unilaterally abort or commit if voted YES
  + must wait
  + commit/abort的决定由TC决定，因此A/B投票YES后必须等待TC
5. When can TC completely forget about a committed transaction?
  + TC在看见全部第四轮ack之后，才能忘记committed transaction
6. When can participant completely forget about a committed transaction?
  + participant在确认TC的commit消息后，能够忘记committed transaction，之后若TC询问，但是没有记录，直接ack

### note

+ Two-phase commit perspective
  + Used in sharded DBs when a transaction uses data on multiple shards
  + cons:
    + slow: multiple rounds of messages
    + slow: disk writes
    + locks are held over the prepare/commit exchanges; blocks other xactions
    + TC crash can cause indefinite blocking, with locks held
  + Thus usually used only in a single small domain
    E.g. not between banks, not between airlines, not over wide area

---

## discussion

1. raft和2PC解决的是不同的问题
  + Raft用于通过备份获取高可用性
    + Raft不能保证所有服务器都完成任务，能保证大多数完成
  + 2PC用于每个参与者完成不同的任务
    + 2PC不能够提升availability，因为所有参与者都必须可获得，整个事务才能成功
2. 同时要求high availability *and* atomic commit
  + TC and servers should each be replicated with Raft
  + run 2PC among replicated services
