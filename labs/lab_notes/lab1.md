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
    seq         int
    state       status
    inputFile   string
    outputFiles []string
    // timer time.Timer
}

type ReduceTask struct {
    seq         int
    state       status
    inputFiles  []string
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

+ 从windows中移动文件到linux，注意需要改变换行符，尤其是脚本文件
+ 为简化，为每个输入文件分配一个map任务。
+ ***rpc传输的数据注意各个filed首字母应该大写导出，否则传递零值***
+ carsh崩溃的情况存在两种
    1. 长时间不响应，master视其为crash
        + 可能之后会向master发送rpc请求
        + 为拒绝其提出的rpc请求，增加序列号机制，每当超时增加序列号，对于就序列号的rpc请求不予执行。
    2. 真正crash
    3. 当这两种结合时，可能存在以下情况：
        + 例如存在三个worker，其中两个在执行map任务，另一个在执行reduce任务，执行reduce任务的不停向master请求读取另外两个生成的结果文件。此时，执行map任务的两个crash了，执行reduce任务的worker死循环等待。
        + 解决：reduce任务请求文件超过一定时间自动结束任务执行，转而执行map任务。


> ***rpc传输的数据注意各个filed首字母应该大写导出，否则传零值***.
