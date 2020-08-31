package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"lab2/src/labrpc"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// Raft peer's state type
//
type state int

//
// Raft peer's state constant
//
const (
	Follower  state = 0
	Candidate state = 1
	Leader    state = 2
)

// states array for debug usage
var states = [3]string{"Follower", "Candidate", "Leader"}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	cond      *sync.Cond          // cond to coordinate goroutines
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	applyCh   chan ApplyMsg       // channel to apply committed message

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// persistent state
	currentTerm int
	log         []Log
	votedFor    int

	// leader elecion volatile state
	state         state
	stateChanged  bool
	lastHeartBeat time.Time
	timeout       int64

	// log replication volatile state
	commitIndex  int
	lastApplied  int
	logApplyCond *sync.Cond

	// log replication leader state
	nextIndex  []int
	matchIndex []int
}

//
// Log entry
//
type Log struct {
	Term    int
	Command interface{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term := rf.currentTerm
	isleader := false
	if rf.state == Leader {
		isleader = true
	}
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term        int
	CandidateID int
	// vote restriction
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// AppendEntries RPC arguments structure
//
type AppendEntriesArgs struct {
	Term     int
	LeaderID int
	// for consistency check
	PrevLogIndex int
	PrevLogTerm  int
	// log entries
	Entries      []Log
	LeaderCommit int
}

//
// AppendEntries RPC reply structure
//
type AppendEntriesReply struct {
	Term    int
	Success bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
	// stale request
	if rf.currentTerm > args.Term {
		return
	}
	// receive larger term, convert to follower
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.state = Follower
		rf.stateChanged = true
		rf.votedFor = -1
		rf.cond.Broadcast()
	}
	// haven't voted yet, grant vote
	// request's log at least up-to-date
	if rf.votedFor == -1 || rf.votedFor == args.CandidateID {
		if checkUptoDate(rf, args.LastLogTerm, args.LastLogIndex) {
			// rf's state
			rf.votedFor = args.CandidateID
			// reply
			reply.Term = rf.currentTerm
			reply.VoteGranted = true
			return
		}
	}
}

// grab lock before calling this function
func checkUptoDate(rf *Raft, term int, lastIndex int) bool {
	if term > rf.log[len(rf.log)-1].Term {
		return true
	}
	if term == rf.log[len(rf.log)-1].Term && lastIndex >= len(rf.log)-1 {
		return true
	}
	return false
}

//
// AppendEntries RPC handler
//
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	log.Printf("[TERM %v] %v %v got heart from term %v leader %v", rf.currentTerm, states[rf.state], rf.me, args.Term, args.LeaderID)
	// stale request
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// advanced request
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = Follower
		rf.stateChanged = true
		rf.cond.Broadcast()
	} else {
		rf.lastHeartBeat = time.Now()
	}
	// return false if log doesn't contain prevLogTerm at prevLogIndex
	if args.PrevLogIndex >= len(rf.log) || rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		// log.Println("log doesn't contain prevLogTerm at prevLogIndex")
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	//
	// exist conflict entry, delete those follow it
	// invariant: log before but not at args.entries[probeIndex] has been synchronized
	//
	probeIndex := 0
	for i := args.PrevLogIndex + 1; i < len(rf.log) && probeIndex < len(args.Entries); i++ {
		log.Printf("[TERM %v] logIndex: %v, rpc args term: %v, log term: %v", rf.currentTerm, i, args.Entries[probeIndex].Term, rf.log[i].Term)
		if args.Entries[probeIndex].Term != rf.log[i].Term {
			log.Printf("[TERM %v]: roll back log result: %v, origin result: %v", rf.currentTerm, rf.log[:i], rf.log)
			rf.log = rf.log[:i]
			break
		}
		probeIndex++
	}
	// append new entries not in the log
	if probeIndex < len(args.Entries) {
		rf.log = append(rf.log, args.Entries[probeIndex:]...)
		probeIndex = len(args.Entries)
	}
	// set commitIndex
	synchronizedIndex := args.PrevLogIndex + probeIndex
	if min(synchronizedIndex, args.LeaderCommit) > rf.commitIndex {
		rf.commitIndex = min(synchronizedIndex, args.LeaderCommit)
		log.Printf("[TERM %v] %v %v set commitIndex to %v", rf.currentTerm, states[rf.state], rf.me, rf.commitIndex)
		rf.logApplyCond.Broadcast()
	}
	reply.Success = true
	reply.Term = rf.currentTerm
}

func min(x int, y int) int {
	if x > y {
		return y
	}
	return x
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := false
	log := Log{}

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Leader {
		return index, term, isLeader
	}

	isLeader = true
	// append new log
	log.Command = command
	log.Term = rf.currentTerm
	rf.log = append(rf.log, log)
	// TODO: persistent
	// get index
	index = len(rf.log) - 1
	// get term
	term = rf.currentTerm

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh

	// Your initialization code here (2A, 2B, 2C).
	rf.mu = sync.Mutex{}
	rf.cond = sync.NewCond(&rf.mu)
	rf.state = Follower
	rf.votedFor = -1
	// log replication
	rf.log = append(rf.log, Log{}) // padding empty log
	rf.logApplyCond = sync.NewCond(&rf.mu)
	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex = append(rf.nextIndex, 1)
		rf.matchIndex = append(rf.matchIndex, 0)
	}
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	// begin raft goroutine
	go raft(rf)
	return rf
}

//
// raft state machine
//
func raft(rf *Raft) {
	// set rand seed
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	// grab lock first so that timeout after follower set initial state
	rf.mu.Lock()
	// timeout goroutine
	go timeoutRoutine(rf)
	// apply log goroutine
	go applyLogRoutine(rf)
	// state machine handle loop
	for {
		switch rf.state {
		case Follower:
			log.Printf("[TERM %v] Follower %v begin\n", rf.currentTerm, rf.me)
			rf.votedFor = -1
			rf.lastHeartBeat = time.Now()
			rf.timeout = int64(360 + r.Intn(360))
		case Candidate:
			log.Printf("[TERM %v] Candidate %v begin \n", rf.currentTerm+1, rf.me)
			rf.currentTerm++
			rf.votedFor = rf.me
			rf.lastHeartBeat = time.Now()
			rf.timeout = int64(360 + r.Intn(360))
			// start vote routine
			go candidateVoteRoutine(rf, rf.currentTerm)
		case Leader:
			log.Printf("[TERM %v] Leader %v begin \n", rf.currentTerm, rf.me)
			// start heartBeatRoutine
			go heartBeatRoutine(rf, rf.currentTerm)
			go appendLogRoutine(rf, rf.currentTerm)
		}
		// wait for state change and reset flag
		for !rf.stateChanged {
			rf.cond.Wait()
		}
		rf.stateChanged = false
	}
}

func appendLogRoutine(rf *Raft, term int) {
	for {
		// wait 10ms to batch append logs
		time.Sleep(10 * time.Millisecond)
		rf.mu.Lock()
		// currently not leader
		if rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
		// send AppendEntreis RPC to each peer
		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			go func(x int) {
				rf.mu.Lock()
				// no log to send
				if rf.nextIndex[x] >= len(rf.log) {
					rf.mu.Unlock()
					return
				}
				rf.mu.Unlock()
				for {
					rf.mu.Lock()
					appendEntriesReply := AppendEntriesReply{}
					appendEntriesArgs := AppendEntriesArgs{
						Term:         term,
						LeaderID:     rf.me,
						PrevLogIndex: rf.nextIndex[x] - 1,
						PrevLogTerm:  rf.log[rf.nextIndex[x]-1].Term,
						Entries:      rf.log[rf.nextIndex[x]:],
						LeaderCommit: rf.commitIndex,
					}
					rf.mu.Unlock()
					for !rf.sendAppendEntries(x, &appendEntriesArgs, &appendEntriesReply) {
						time.Sleep(10 * time.Millisecond)
						rf.mu.Lock()
						// term has changed
						if rf.currentTerm != term {
							rf.mu.Unlock()
							return
						}
						rf.mu.Unlock()
					}
					rf.mu.Lock()
					// stale reply
					if term != rf.currentTerm {
						rf.mu.Unlock()
						return
					}
					// reply larger term
					if appendEntriesReply.Term > rf.currentTerm {
						rf.state = Follower
						rf.stateChanged = true
						rf.currentTerm = appendEntriesReply.Term
						rf.cond.Broadcast()
						rf.mu.Unlock()
						return
					}
					// success reply
					if appendEntriesReply.Success {
						rf.matchIndex[x] = appendEntriesArgs.PrevLogIndex + len(appendEntriesArgs.Entries)
						rf.nextIndex[x] = rf.matchIndex[x] + 1
						// increase commitIndex
						lastIndex := appendEntriesArgs.PrevLogIndex + len(appendEntriesArgs.Entries)
						if rf.log[lastIndex].Term == rf.currentTerm && lastIndex > rf.commitIndex {
							count := 1
							for i := 0; i < len(rf.peers); i++ {
								if i == rf.me {
									continue
								}
								if rf.matchIndex[i] >= lastIndex {
									count++
									if count > len(rf.peers)/2 {
										rf.commitIndex = lastIndex
										rf.logApplyCond.Broadcast()
										break
									}
								}
							}
						}
						rf.mu.Unlock()
						return
					}
					// TODO: Fast Roll Back
					log.Printf("[TERM %v] %v %v Append %v Failed nextIndex: %v", rf.currentTerm, states[rf.state], rf.me, x, rf.nextIndex[x])
					rf.nextIndex[x]--
					// hold lock in order to construct new RPC request
					rf.mu.Unlock()
				}
			}(i)
		}
	}
}

func applyLogRoutine(rf *Raft) {
	for {
		rf.logApplyCond.L.Lock()
		for !rf.logApplyEnable() {
			rf.logApplyCond.Wait()
		}
		// get commitIndex
		commitIndex := rf.commitIndex
		// release lock in case channel block
		rf.logApplyCond.L.Unlock()
		for i := rf.lastApplied + 1; i <= commitIndex; i++ {
			applyMsg := ApplyMsg{
				CommandValid: true,
				Command:      rf.log[i].Command,
				CommandIndex: i,
			}
			// log.Printf("trigger log apply!! index: %v", i)
			rf.applyCh <- applyMsg
			rf.lastApplied++
		}
	}
}

//
// check whether able to apply new log
// grab lock before calling this func
//
func (rf *Raft) logApplyEnable() bool {
	if rf.commitIndex > rf.lastApplied {
		return true
	}
	return false
}

func timeoutRoutine(rf *Raft) {
	for {
		// sleep 10ms
		time.Sleep(10 * time.Millisecond)
		// get rf lock to access lastHeartBeat
		rf.mu.Lock()
		if rf.state == Leader {
			rf.mu.Unlock()
			continue
		}
		// state has been changed in current term
		if rf.stateChanged {
			rf.mu.Unlock()
			continue
		}
		// trigger timeout
		// log.Printf("[TERM %v] %v %v timeout: %v, past: %v\n", rf.currentTerm, states[rf.state], rf.me, rf.timeout, time.Since(rf.lastHeartBeat).Milliseconds())
		if time.Since(rf.lastHeartBeat).Milliseconds() > rf.timeout {
			// log.Printf("[TERM %v] %v %v timeout triggered\n", rf.currentTerm, states[rf.state], rf.me)
			rf.state = Candidate
			rf.stateChanged = true
			rf.cond.Broadcast()
		}
		rf.mu.Unlock()
	}
}

func candidateVoteRoutine(rf *Raft, term int) {
	mu := sync.Mutex{}
	voteCount := 1
	// start vote goroutine for each client
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		go func(x int) {
			requestVoteReply := RequestVoteReply{}
			requestVoteArgs := RequestVoteArgs{
				CandidateID:  rf.me,
				Term:         term,
				LastLogIndex: len(rf.log) - 1,
				LastLogTerm:  rf.log[len(rf.log)-1].Term,
			}
			for !rf.sendRequestVote(x,
				&requestVoteArgs,
				&requestVoteReply) {
				time.Sleep(10 * time.Millisecond)
			}
			rf.mu.Lock()
			defer rf.mu.Unlock()
			// stale reply
			if term != rf.currentTerm || rf.stateChanged || rf.state == Leader {
				return
			}
			// reply larger term exists
			if requestVoteReply.Term > rf.currentTerm {
				rf.state = Follower
				rf.stateChanged = true
				rf.currentTerm = requestVoteReply.Term
				rf.cond.Broadcast()
			}
			// failed vote
			if !requestVoteReply.VoteGranted {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			voteCount++
			// got majority votes
			log.Printf("[TERM %v] %v %v Got vote from %v\n", rf.currentTerm, states[rf.state], rf.me, term)
			if voteCount > len(rf.peers)/2 {
				rf.state = Leader
				rf.stateChanged = true
				rf.cond.Broadcast()
			}
		}(i)
	}
}

func heartBeatRoutine(rf *Raft, term int) {
	// send heartBeat to each client every 120ms, retry if failed
	for {
		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			go func(x int) {
				rf.mu.Lock()
				appendEntriesArgs := AppendEntriesArgs{
					LeaderID:     rf.me,
					Term:         term,
					PrevLogIndex: rf.nextIndex[x] - 1,
					PrevLogTerm:  rf.log[rf.nextIndex[x]-1].Term,
					LeaderCommit: rf.commitIndex,
				}
				rf.mu.Unlock()
				appendEntriesReply := AppendEntriesReply{}
				for !rf.sendAppendEntries(x, &appendEntriesArgs, &appendEntriesReply) {
					time.Sleep(10 * time.Millisecond)
					rf.mu.Lock()
					if rf.currentTerm != term {
						rf.mu.Unlock()
						return
					}
					rf.mu.Unlock()
				}
			}(i)
		}
		time.Sleep(120 * time.Millisecond)
		rf.mu.Lock()
		// term has changed
		if rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
	}
}
