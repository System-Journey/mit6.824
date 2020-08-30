# lab2: Raft

## raft API

raft库向上层应用导出一个接口，有以下方法：

```go
//
// 创建一个新的raft服务器实例
// peers: 网络识别符数组
// me: 当前peer
// 
rf := Make(peers, me, persister, applyCh)
//
// 开启对一个新log的共识过程
// Start()应当立刻返回，不等待log append完成
//
rf.Start(command interface{}) (index, term, isleader)
//
// 询问一个raft服务器其当前term，以及是否是leader
//
rf.GetState() (term, isLeader)
//
// 每次提交一个新log entry，每一个Raft peer应该通过applyCh向上层应用发送一个ApplyMsg
//
type ApplyMsg
```

### lab 2A Design Document

实现Raft leader election和heartbeats。goal：

1. a single leader to be elected
2. leader remains to be the leader if no failures
3. a new leader can take over if the old leader fails or if packets to/from the old leader are lost

#### 结合skeleton code的leader选举过程回顾

1. `Make()`创建peer实例，随机设置超时时间，进入follower状态。
2. 如果收到heartbeats
    + 验证
    + 保持follower
3. 如果收到requestVote
    + 验证
    + 投票
4. 如果超时，term++，设置随机定时器，发送`RequestVote`rpc，进入candidate状态
    + 收到过半数vote，说明赢得选举，开始发送heartbeat
    + 收到term>=自己的heartbeats或者>自己的RequestVote，说明选举失败。转为follower。否则拒绝。
    + 超时，term++，随机设置超时时间，继续开启选举。

#### 数据结构

```go

type state int

const (
    Follower state = 0
    Candidate state = 1
    Leader state = 2
)

type Raft struct {
    cond      *sync.Cond          // cond to coordinate goroutines
    // persistent state
	currentTerm int
	log         []Log
	votedFor    int

	// volatile state
	state         state
	stateChanged  bool
	lastHeartBeat time.Time
	timeout       int64
}

type Log struct {

}

type RequestVoteArgs struct {
    Term int
    CandidateID int
}

type RequestVoteReply struct {
    Term int
    voteGranted bool
}

type AppendEntriesArgs struct {
    Term int
    LeaderID int
}

type AppendEntriesReply struct {
    Term int
    Success bool
}
```

#### 算法

1. raft状态机线程
    + 状态机，三个状态执行不同的操作，当发生状态转移时，由其它子线程通知。
    + follower
        + 初始化timeout
        + 开启timeout go routine
        + 等在channel上
    + candidate
        + currentTerm++
        + 给自己投票
        + 重置timeout
        + 每个client独立的发送voteRequest gorodutine
        + 开启timeout go routine
        + 等在channel上
    + leader
        + 开启heartbeat发送线程
        + 等在channel上
2. AppendEntriesRPC
    + server收到回复:
        1. 比较发送时term和当前term
    + client：figure 2
3. RequestVoteRPC
    + server收到回复:
        1. 比较发送时term和当前term
        2. signal条件变量，获得vote数加一
    + client: figure 2

注意

+ 接收到RPC和发送时term不一致
+ 接收到RPC时和发送时raft不一致
+ heartbeat和正常appendEntry的处理

#### 同步

+ 利用cond条件变量同步主raft线程和子线程
+ 子线程或者rpc响应改变了当前peer的状态时，激活主raft线程中的条件变量

#### 复盘

+ 子routine回传信息给父routine不好处理时，考虑使用共享内存，并尝试在子routine内就地解决，例如接收超过半数vote可以在子线程内递增闭包变量判断，并改变peer的状态，不用在父线程中等待
+ 出现测试有时能通过有时无法通过的情况，循环跑多次
+ trace首先输出逻辑时钟（当前term）较为方便`log.Printf("[TERM %v]...", rf.currentTerm)`
+ timeout线程不能返回
