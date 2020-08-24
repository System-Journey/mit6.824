# Design Document

## MapReduce过程结合skeleton代码的理解

1. Master启动
2. Worker启动，向Master发送RPC请求分配任务
3. Master响应未完成的任务的文件名
4. Worker读取文件，调用Map函数，写入本地R个文件
5. Worker通过RPC通知Master map任务的完成
6. 若仍存在reduce任务，Master以reduce任务作为响应
7. Reduce Worker通过RPC询问Map结果文件地址
8. Master将文件地址作为响应或者告知可以开始reduce任务
9. Reduce Worker对中间键值对进行排序(若中间文件过大，需要外部排序)
10. Reduce Worker执行Reduce函数，写入本地输出文件
11. Reduce Worker通知Master执行完成

## 数据结构和函数

### Master数据结构

```go
// Master
type Master struct {
    nWorker int32
    nReducer int32
    mu sync.Mutex
    mapTasks []MapTask
    reduceTasks []ReduceTask
}

// Task Status
type status int32

const (
	statusIdle       status = 0
	statusInprogress status = 1
	statusCompleted  status = 2
)

type MapTask struct {
    state status
    inputFile string
    outputFiles []string
    // timer time.Timer
}

type ReduceTask struct {
    state status
    inputFiles []string
    // timer time.Timter
}
```

### rpc handlers

#### `MapTaskRequest`

#### `ReduceTaskRequest`

## 算法

### Worker崩溃

+ 分配Map/Reduce任务后，若10s没有收到响应，则说明Worker失败
    + `time.Timer`或者
    + `time.sleep(10 * time.Second)`
+ 重新分配给新Worker
    + 此时没有Worker，等待
+ 垃圾回收：
    + 不需要，直接分配就好

## 同步

Master并发处理RPC请求，对其数据结构的访问需要加锁。

## 复盘

+ 为简化，为每个输入文件分配一个map任务。
