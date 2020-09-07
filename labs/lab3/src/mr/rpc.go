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
	Seq       int
	TaskID    string
	Inputfile string
	NReduce   int
	Valid     bool
}

type MapFinishArgs struct {
	Seq         int
	TaskID      string
	Outputfiles []string
	Rst         bool
}

// reduce task

type ReduceTaskReply struct {
	Seq     int
	TaskID  string
	NWorker int
	Valid   bool
}

type ReduceFileArgs struct {
	Seq    int
	TaskID string
}

type ReduceFileReply struct {
	Intermediatefiles []string
	Rst               bool
}

type ReduceFinishArgs struct {
	Seq    int
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
