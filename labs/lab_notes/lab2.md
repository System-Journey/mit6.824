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
    Term int
    Command interface{}
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
        + 等在cond上
    + candidate
        + currentTerm++
        + 给自己投票
        + 重置timeout
        + 每个client独立的发送voteRequest gorodutine
        + 开启timeout go routine
        + 等在cond上
    + leader
        + 开启heartbeat发送线程
        + 等在cond上
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
+ 当子线程或者rpc响应改变了当前peer的状态时，激活主raft线程中的条件变量

#### 复盘

+ 子routine回传信息给父routine不好处理时，考虑使用共享内存，并尝试在子routine内就地解决，例如接收超过半数vote可以在子线程内递增闭包变量判断，并改变peer的状态，不用在父线程中等待
+ 出现测试有时能通过有时无法通过的情况，循环跑多次
+ trace首先输出逻辑时钟（当前term）较为方便`log.Printf("[TERM %v]...", rf.currentTerm)`
+ timeout线程不能返回

### lab 2B Design Document

goal：实现append新log的agreement。

#### 结合skeleton code的append entries流程回顾

+ 专用appendLog线程
+ 专用applyLog线程

1. 上层应用通过`Start()`发送需要commit的命令。
2. leader将log加入自己状态，持久化。
3. appendLog触发时开始agreement处理。
4. leader向各个follower发送AppendEntries RPC。
5. 如果成功，`argeeCount`递增，如果失败，回滚`nextIndex`。
6. 超过半数，agreement成功，向`applyCh`中发送命令。

#### 数据结构

```go
type Raft struct {
    applyCh    chan ApplyMsg      // channel to send committed message
    cond      *sync.Cond          // cond to coordinate goroutines
    // persistent state
	currentTerm int
	log         []Log
	votedFor    int
	// leader election volatile state
	state         state
	stateChanged  bool
	lastHeartBeat time.Time
    timeout       int64
    // log replication volatile state
    commitIndex   int
    lastApplied   int
    // log replication leader state
    nextIndex     []int // 初始化为最后一个log索引+1
    matchIndex    []int // 每个服务器已知的成功commit最大的索引
}

type AppendEntriesArgs struct {
    Term int
    LeaderID int
    // consistency check
    PrevLogIndex int
    PrevLogTerm  int
    // log entries
    Entries []Log
    LeaderCommit int
}

type AppendEntriesReply struct {
    Term int
    Success bool
}

type RequestVoteArgs struct {
    Term int
    CandidateID int
    // vote restriction
    LastLogIndex int
    LastLogTerm  int
}
```

#### 算法

##### 单leader log共识

1. leader主线程
    + 每个server对应开启一个RPC goroutine处理
2. leader RPC发送线程处理返回值
    + false：重试
    + term不正确，转为follower
    + success为false，回滚日志重试
3. follower RPC回调: figure2

##### leader选举约束

+ 需要`lastLogTerm`更大或者`lastLogTerm`相同`lastLogIndex`更大

##### 更新本地commit Index的条件

1. 和leader同步过
2. leader处该log已经提交
3. 该Index比本地commitIndex大

#### 同步

1. 独立的command apply线程以防send操作阻塞，向`applyCh`中顺序发送log entry。(从lastApplied发送到commitIndex)

#### 复盘

1. 隐藏bug: 对于voteRequest请求接收者，如其term小于请求的candiadate所在的term，需要转变为follower，此时若直接给予投票，则主状态机循环中就不应当设置votedFor为-1，否则可能出现同一server同一term投两次票的情况。
    + 解决：转变状态为follower的同时即设置votedFor为-1，而不再在主状态机中设置votedFor。
