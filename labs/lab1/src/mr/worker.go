package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

var (
	mapfun    func(string, string) []KeyValue
	reducefun func(string, []string) string
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the master.
	// CallExample()
	mapfun = mapf
	reducefun = reducef

	for {
		// request map task
		mtReply := CallMapTaskRequest()
		// map task received
		if mtReply.Valid {
			ofiles := doMap(mtReply)
			// notify master
			CallMapFinish(mtReply.TaskID, ofiles)
			continue
		}

		// request reduce task
		rdReply := CallReduceTaskRequest()
		// reduce task received
		if rdReply.Valid {
			doReduce(rdReply)
			// notify master
			CallReduceFinish(rdReply.TaskID)
			continue
		}
	}
}

//
// CallMapTaskRequest to master
//
func CallMapTaskRequest() MapTaskReply {
	args := EmptyArgs{}
	reply := MapTaskReply{}

	call("Master.MapTaskRequest", &args, &reply)

	if reply.Valid {
		// log.Println("[MAP WORKER " + reply.TaskID + "] start")
	}
	return reply
}

//
// CallMapFinish to master
//
func CallMapFinish(taskID string, ofiles []string) {
	args := MapFinishArgs{}
	reply := EmptyReply{}

	args.TaskID = taskID
	args.Outputfiles = ofiles
	call("Master.MapTaskFinish", &args, &reply)
	// log.Println("[MAP WORKER " + taskID + "] complete")
}

//
// CallReduceTaskRequest to master
//
func CallReduceTaskRequest() ReduceTaskReply {
	args := EmptyArgs{}
	reply := ReduceTaskReply{}

	call("Master.ReduceTaskRequest", &args, &reply)

	if reply.Valid {
		// log.Println("[REDUCE WORKER " + reply.TaskID + "] start")
	}
	return reply
}

//
// CallReduceFileRequest to master
//
func CallReduceFileRequest(taskID string) []string {
	// log.Printf("[REDUCE WORKER %v] request file\n", taskID)
	args := ReduceFileArgs{}
	reply := ReduceFileReply{}

	args.TaskID = taskID
	call("Master.ReduceFileRequest", &args, &reply)

	return reply.Intermediatefiles
}

//
// CallReduceFinish to master
//
func CallReduceFinish(taskID string) {
	args := ReduceFinishArgs{}
	reply := EmptyReply{}

	args.TaskID = taskID

	call("Master.ReduceTaskFinish", &args, &reply)
	// log.Println("[REDUCE WORKER " + taskID + "] finish")
}

//
// example function to show how to make an RPC call to the master.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Master.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

func doMap(reply MapTaskReply) []string {
	// read input data
	file, err := os.Open(reply.Inputfile)
	if err != nil {
		log.Fatalf("cannot open %v", reply.Inputfile)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v, filename", file.Name())
	}
	file.Close()
	// calc intermediate key value pairs
	kva := mapfun(reply.Inputfile, string(content))
	// write r intermediate files
	interfs := []*os.File{}
	encoders := []*json.Encoder{}
	for i := 0; i < reply.NReduce; i++ {
		dir, _ := os.Getwd()
		f, _ := ioutil.TempFile(dir, "*")
		interfs = append(interfs, f)
		encoders = append(encoders, json.NewEncoder(f))
	}
	for _, kv := range kva {
		i := ihash(kv.Key) % reply.NReduce
		encoders[i].Encode(&kv)
	}
	// rename temporary files
	paths := []string{}
	for i, f := range interfs {
		dir, _ := os.Getwd()
		oname := "mr-" + reply.TaskID + "-" + strconv.Itoa(i)
		newPath := filepath.Join(dir, oname)
		err := os.Rename(f.Name(), newPath)
		if err != nil {
			log.Fatal("[MAP WORKER" + reply.TaskID + "] rename temporary files failed")
		}
		paths = append(paths, newPath)
	}
	// return file paths
	return paths
}

func doReduce(reply ReduceTaskReply) {
	intermediate := []KeyValue{}
	i := 0
	for i < reply.NWorker {
		fs := CallReduceFileRequest(reply.TaskID)
		for _, filename := range fs {
			// log.Printf("[REDUCE WORKER "+reply.TaskID+"] get file %v\n", filename)
			file, err := os.Open(filename)
			if err != nil {
				log.Fatalf("cannot open %v", filename)
			}
			dec := json.NewDecoder(file)
			for {
				kv := KeyValue{}
				if err := dec.Decode(&kv); err != nil {
					break
				}
				intermediate = append(intermediate, kv)
			}
			file.Close()
			i++
		}
	}
	// sorting intermediate files
	sort.Sort(ByKey(intermediate))
	// reduce on intermediate files
	i = 0
	dir, _ := os.Getwd()
	tempFile, _ := ioutil.TempFile(dir, "*")
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducefun(intermediate[i].Key, values)

		fmt.Fprintf(tempFile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	tempFile.Close()
	oname := "mr-out-" + reply.TaskID
	os.Rename(tempFile.Name(), filepath.Join(dir, oname))
}
