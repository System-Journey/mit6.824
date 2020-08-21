# lec2 RPC and Threads

## threads

### motivation

+ I/O concurrency
+ parallelism
+ convenient reasons

#### OR event-driven programming (asynchronize programming)  

1. one loop, one thread, less cost than threads
2. I/O concurrency
3. kind of hard to harness multi core  

### thread challenge

1. race
2. coordination
    + channels
    + condition variables
    + wait group
3. deadlock

#### crawler example

> go中map本质上是一个指针，因此传map即传map的引用。  
> `go run --race ..go`可以检测race，运行时分配shadow memory并且check读写，非静态检测。

+ Exploit I/O concurrency
+ Fetch each URL only *once*

1. style1: shared memory
    ```go
    type fetchState struct {
	    mu      sync.Mutex
	    fetched map[string]bool
    }

    func concurrentMutex(...) {
        // check if url crawlered 
        f.mu.Lock()
    	already := f.fetched[url]
    	f.fetched[url] = true
    	f.mu.Unlock()
    	if already {
    		return
        }
        // use waitgroup to coordinate multiple threads
        var done sync.WaitGroup
	    for _, u := range urls {
            done.Add(1)
            // 1. u迭代变量始终在变化，可以重新定义一个闭包变量u2
            u2 := u
	    	go func() {
	    		defer done.Done()
	    		ConcurrentMutex(u2, fetcher, f)
            }()
            // 2. 传值
	    	//go func(u string) {
	    	//	defer done.Done()
	    	//	ConcurrentMutex(u, fetcher, f)
	    	//}(u)
	    }
	    done.Wait()
    }
    ```
    + `waitgroup`本质上是counter，用于线程的同步。
    + defect: 创建了太多的线程，可以利用线程池。
2. style2: using channel
    ```go
    // worker向channel中发送urls
    func worker(url string, ch chan []string, fetcher Fetcher) {
    	urls, err := fetcher.Fetch(url)
    	if err != nil {
    		ch <- []string{}
    	} else {
    		ch <- urls
    	}
    }
    // master从channel中接受urls
    func master(ch chan []string, fetcher Fetcher) {
    	n := 1
    	fetched := make(map[string]bool)
    	for urls := range ch {
    		for _, u := range urls {
    			if fetched[u] == false {
    				fetched[u] = true
    				n += 1
    				go worker(u, ch, fetcher)
    			}
    		}
    		n -= 1
    		if n == 0 {
    			break
    		}
    	}
    }

    func ConcurrentChannel(url string, fetcher Fetcher) {
    	ch := make(chan []string)
    	go func() {
    		ch <- []string{url}
    	}()
    	master(ch, fetcher)
    }
    ```

#### when to use sharing and locks, versus channels?

+ sharing and locks: **state**
+ channels: **communication**
+ sharing+locks for state, and sync.Cond or channels or time.Sleep() for waiting/notification

## RPC

+ goal: easy-to-program client/server communication
+ client sends request
+ server gives response

``` go
// software structure
client app          handler fns
 stub fns            dispatcher
 RPC lib             RPC lib
   net                  net
```

### kv

```go
package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

//
// Common RPC request/reply definitions
//

const (
	OK       = "OK"
	ErrNoKey = "ErrNoKey"
)

type Err string

type PutArgs struct {
	Key   string
	Value string
}

type PutReply struct {
	Err Err
}

type GetArgs struct {
	Key string
}

type GetReply struct {
	Err   Err
	Value string
}

//
// Client
//

func connect() *rpc.Client {
	client, err := rpc.Dial("tcp", ":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return client
}

// get stub
func get(key string) string {
	client := connect()
	args := GetArgs{"subject"}
	reply := GetReply{}
	err := client.Call("KV.Get", &args, &reply)
	if err != nil {
		log.Fatal("error:", err)
	}
	client.Close()
	return reply.Value
}

// put stub
func put(key string, val string) {
	client := connect()
	args := PutArgs{"subject", "6.824"}
	reply := PutReply{}
	err := client.Call("KV.Put", &args, &reply)
	if err != nil {
		log.Fatal("error:", err)
	}
	client.Close()
}

//
// Server
//

type KV struct {
	mu   sync.Mutex
	data map[string]string
}

func server() {
	kv := new(KV)
	kv.data = map[string]string{}
	rpcs := rpc.NewServer()
	rpcs.Register(kv)
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err == nil {
				go rpcs.ServeConn(conn)
			} else {
				break
			}
		}
		l.Close()
	}()
}

func (kv *KV) Get(args *GetArgs, reply *GetReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	val, ok := kv.data[args.Key]
	if ok {
		reply.Err = OK
		reply.Value = val
	} else {
		reply.Err = ErrNoKey
		reply.Value = ""
	}
	return nil
}

func (kv *KV) Put(args *PutArgs, reply *PutReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.data[args.Key] = args.Value
	reply.Err = OK
	return nil
}

//
// main
//

func main() {
	server()

	put("subject", "6.824")
	fmt.Printf("Put(subject, 6.824) done\n")
	fmt.Printf("get(subject) -> %s\n", get("subject"))
}
```

+ *Client*
  + `connect()`'s `Dial()` creates a TCP connection to the server
  + `get()` and `put()` are client "stubs
  + `Call()`执行rpc调用
    + 需要明确服务器函数名，rpc调用参数，rpc调用的返回值
    + rpc库marshall参数，发送请求，等待，unmarshall响应，`Call()`返回值表明是否接收到响应。Err表明服务级别故障。
+ *Server*
  + 服务器注册
    1. 服务器声明一个具有方法的对象作为RPC的handler。
    2. 服务器注册该对象到RPC库。
    3. 服务器接受TCP连接，转发给RPC库。
  + RPC库
    1. 读取每一个请求
    2. 为每一个请求创建一个新goroutine
    3. unmarshall请求
    4. 查询named object
    5. 调用object的named method
    6. marshall响应
    7. 向TCP连接写响应
  + 服务器的`Get()`和`Put()`handlers
    + 必须加锁，因为RPC库为每一个新请求创建一个新goroutine
    + 读rpc参数，修改rpc响应

#### details

+ *Binding*: 客户端如何知道通信的服务器？
  + 服务器的name/port是`Dial()`的参数
+ *Marshalling*：format data into packets
  + go的RPC库能够传递`strings, arrays, objects, maps, &c`
  + go通过拷贝指针指向的数据传指针
  + 无法传递channel或函数

### RPC故障

#### client view

1. 客户端无法接受服务器的响应
2. 客户端无法知道服务器是否接收到请求
    + 服务器未收到请求
    + 服务器执行并在发送前崩溃
    + 服务器执行并在发送前网络中断

#### 故障处理机制

##### best effort

> `Call()`等待响应，若未收到，则重发请求。重复多次后返回error。

+ 适用情况
    + 只读操作
    + 重复操作无影响

##### at most once

idea在于服务器RPC代码检测重复请求，返回之前的响应而非re-running handler。

检测的方法要求客户端为每一个请求添加一个唯一的ID号(XID)，当re-send时使用相同的ID。

```python
server:
    if seen[xid]:
        r = old[xid]
    else
        r = handler()
        old[xid] = r
        seen[xid] = true
```

1. Q：如何保证不同客户端使用不同XID?
    + 结合唯一的client ID(ip address?)和序列号
2. Q：服务器何时可以丢弃旧响应?
    + 客户端的每次RPC包含ACK信息确认`<= X`的请求已接受
    + 仅允许客户端最多存在一个rpc调用，则seq+1调用到达意味着服务器可以丢弃`<=seq`的请求
3. Q：在原始响应仍在处理时如何处理重复响应？
    + 为执行RPC分配“pending”标记，等待或者忽略
4. Q：at-most-once服务器崩溃然后重启时怎么办？
    + WAL
    + 备份服务器同时记录背负信息

+ Go的实现是一种简单形式的"at-most-once"
    + Go rpc不重发请求，因此服务器不可见重复请求
    + 未得到响应时返回error
        + 超时(from TCP)
        + 服务器没有收到请求
        + 服务器处理了请求，但是服务器/网络在返回响应前崩溃

##### exactly once

无限重发+重复检测+容错机制。
