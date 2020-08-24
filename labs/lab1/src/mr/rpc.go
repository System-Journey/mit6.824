package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type EmptyArgs struct {
}

type EmptyReply struct {
}

// map task
type MapTaskReply struct {
	TaskID    string
	Inputfile string
	NReduce   int
	Valid     bool
}

type MapFinishArgs struct {
	TaskID      string
	outputfiles []string
}

// reduce task

type ReduceTaskReply struct {
	TaskID  string
	NWorker int
	Valid   bool
}

type ReduceFileArgs struct {
	TaskID string
}

type ReduceFileReply struct {
	intermediatefiles []string
}

type ReduceFinishArgs struct {
	TaskID string
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
