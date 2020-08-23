# lec5 Go, Threads, and Raft

## pattern 1: 循环多线程创建和闭包

```go
func main() {
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(x int) {
            sendRPC(x)
            wg.Done()
        }(i)
        // buggy code: i在循环中不断改变
        // go func() {
        //  sendRPC(i)
        //  wg.Done()
        // }()
    }
    wg.Wait()
}
```

## pattern 2: 周期性执行

+ `time.Sleep(), time.Now()`
+ heartbeat

```go
for !rf.killed() {
    println()
}
```

## pattern 3: defer的使用

获得锁之后，利用defer语句接着释放锁。

```go
mu.Lock()
defer mu.Unlock()
```

## pattern 4: lock 

### rule

1. every time access shared data, grab a lock
2. locks also protect invariants/data integrity

> Locks make regions of code atomic, not just single statement or single update to shared variables.

## pattern 5: cond variables

> Cond variable is a queue of threads waiting for something inside a critical section. Key idea is to make it possible to go to sleep inside critical section by atomically releasing lock at time we go to sleep.

1. wait: Atomically release lock and go to sleep. Re-acquire lock later, before returning.

## pattern 6: channel

+ unbuffered channel => sender and receiver all wait for each other
+ use of channels:
    + collect data from goroutines
    + replace wait group
