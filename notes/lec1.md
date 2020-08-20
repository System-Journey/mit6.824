# lec1 Introduction

## preview

### goal

+ **parallelism**
+ **fault tolerance**
+ physical
+ security/isolated

### challenges

+ concurrency
+ partial failure
+ performance

### Infrastructure

+ **storage**
+ communication
+ computation

> Non-distributed Abstractions hide underlying distributed facts.

#### Implementation tools

+ RPC
+ threads
+ concurrency control: locks

#### Topic 1: Performance

+ scalable speed up: **scalability**
  + linear performance approvement

#### Topic 2: Fault Tolerance

+ **availability**
+ **recoverablity**

##### tools

+ NV storage
+ Replication

#### Topic 3: Consistency

+ strong consistency
  + communication cost
+ weak consistency

## MapReduce framework

```rust
Input 1 -> Map |(a, 1)|(b, 1)|      |  -> reduce a -> (a, 2)
Input 2 -> Map |      |(b, 1)|      |  -> reduce b -> (b, 2)
Input 3 -> Map |(a, 1)|      |(c, 1)|  -> reduce c -> (c, 1)

```

+ *shuffle*: map worker输出单行(k,v)对，同一列(k,v)对作为reduce worker的输入。
+ GFS会自动将文件划分为64MB chunk进行存储。

```python
Map(k, v)
    split v into word
    for each word w
        emit(w, 1)

# v is a vector of words
Reduce(k, v)
    emit(len(v))
```

### trick

1. master调度时为存储输入文件的服务器分配map任务。

### limited

1. 行列shuffle时的网络通信。
2. output结果写入gfs的网络通信。
