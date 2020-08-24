package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

type status int

const (
	statusIdle       status = 0
	statusInprogress status = 1
	statusCompleted  status = 2
)

type Master struct {
	// Your definitions here.
	nWorker     int
	nReducer    int
	mu          sync.Mutex
	mapTasks    []MapTask
	reduceTasks []ReduceTask
}

type MapTask struct {
	state       status
	inputFile   string
	outputFiles []string
}

type ReduceTask struct {
	state      status
	inputFiles map[int]string
}

// Your code here -- RPC handlers for the worker to call.

//
// MapTaskRequest handler, ARGS unused
//
func (m *Master) MapTaskRequest(args *EmptyArgs, reply *MapTaskReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.mapTasks {
		mt := &m.mapTasks[i]
		if mt.state == statusIdle {
			mt.state = statusInprogress
			go func() {
				time.Sleep(10 * time.Second)
				if mt.state != statusCompleted {
					mt.state = statusIdle
				}
			}()
			reply.Inputfile = mt.inputFile
			reply.Valid = true
			reply.NReduce = m.nReducer
			reply.TaskID = strconv.Itoa(i)
			break
		} else {
			reply.Valid = false
		}
	}
	return nil
}

//
// MapFinshHandler is handler for map task finish call
//
func (m *Master) MapFinishHandler(args *MapFinishArgs, reply *EmptyReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// fullfill maptask struct
	id, _ := strconv.Atoi(args.TaskID)
	mt := m.mapTasks[id]
	mt.state = statusCompleted
	mt.outputFiles = args.outputfiles
	return nil
}

//
// ReduceTask request handler
//
func (m *Master) ReduceTaskRequest(args *EmptyArgs, reply *ReduceTaskReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.reduceTasks {
		rt := m.reduceTasks[i]
		if rt.state = statusIdle {
			rt.state = statusInprogress
			go func() {
				time.Sleep(10 * time.Second)
				if rt.state != statusCompleted {
					// reset reduce task state
					rt.state = statusIdle
					rt.inputFiles = map[int]string{}
				}
			}()
			reply.NWorker = m.nWorker
			reply.TaskID  = strconv.Itoa(i)
			reply.Valid   = true
		} else {
			reply.Valid = false;
		}
	}
	return nil
}


//
// ReduceFileRequest to master
//
func (m *Master) ReduceFileRequest(args *ReduceFileArgs, reply *ReduceFileReply) error {
	id := strconv.Atoi(args.TaskID)
	m.mu.Lock()
	defer m.mu.Unlock()
	reply.intermediatefiles = []string{}
	replyfiles := &reply.intermediatefiles
	for i := 0; i < m.nWorker; i++ {
		if _, ok := m.reduceTasks[id].inputFiles; ok == false && m.mapTasks[i].state == statusCompleted {
			replyfiles = append(replyfiles, m.mapTasks[i].outputFiles[id])
		}
	}
	return nil
}

//
// ReduceTaskFinish to master
//
func (m *Master) ReduceTaskFinish(args *ReduceFinishArgs, reply *EmptyReply) error {
	id := strconv.Atoi(args.TaskID)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reduceTasks[id].state = statusCompleted
	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false

	// Your code here.

	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.

	m.nWorker = len(files)
	m.nReducer = nReduce
	m.mu = sync.Mutex{}
	m.mapTasks = []MapTask{}
	m.reduceTasks = []ReduceTask{}

	for _, f := range files {
		mt := MapTask{}
		mt.inputFile = f
		mt.outputFiles = []string{}
		m.mapTasks = append(m.mapTasks, mt)
	}

	for i := 0; i < nReduce; i++ {
		rt := ReduceTask{}
		rt.inputFiles = []string{}
		m.reduceTasks = append(m.reduceTasks, rt)
	}

	m.server()
	return &m
}
