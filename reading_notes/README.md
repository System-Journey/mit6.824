# reading notes

- [x] [MapReduce: Simplified Data Processing on Large Clusters](./1.mapreduce.md)
- [x] [The Google File System](./2.gfs.md)
- [x] [The Design of a Practical System for Fault-Toloerant Virtual Machines](./3.vm-ft.md)
- [x] [The Go memory model](./4.go-memory-model.md)
- [x] [In Search of an Understandable Consensus Algorithm I](./5.raft-I.md)
- [x] [In Search of an Understandable Consensus Algorithm II](./6.raft-II.md)
- [x] [ZooKeeper: Wait free coordination for Internet-scale systems](./7.zookeeper.md)
- [x] [Object Storage on CRAQ: High-throughput chain replication for read-mostly workloads](./8.craq.md)
- [x] [Amazon Aurora: Design Considerations for High Throughput Clound-Native Relational Databases](./9.aurora.md)

## How to read a paper

### first pass

#### steps

1. 阅读title，abstraction和introduction
2. 阅读section和sub-section标题
3. 扫读数学内容，确定理论基础
4. 阅读conclusion
5. 查看引用

#### goal

能够回答以下五个问题

1. *Category*: What type of paper is this? A measurement paper? An analysis of an existing system? A description of a research prototype?
2. *Context*: Which other papers is it related to? Which theoretical bases were used to analyze the problem?
3. *Correctness*: Do the assumptions appear to be valid?
4. *Contributions*: What are the paper’s main contributions?
5. *Clarity*: Is the paper well written?

### second pass

#### steps

+ 细致阅读，但是忽略例如证明的细节。
+ 在旁白记录不理解的概念、问题
+ 注意图表
+ 标记paper中出现的引用

#### goal

+ 能够理解paper中每一幅图和表格。
+ 能够总结文章的整体框架
+ 能够理解文章的主旨和贡献

### third pass

#### steps

+ 细致阅读，尝试解决pass-2中的问题
+ 分治paper的贡献，一一理解

#### goal

+ 基于自己的理解、旁白记的笔记、pass-2的文章框架，输出文章的笔记
+ 同时提出问题

### fourth pass

+ lecture note预习
+ lecture视频并补充笔记
+ lecuture note并补充笔记
+ 论文阅读复盘
