// This package maintains the transition of states for servers.
package logic

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/iketheadore/raft/comm"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// logic control module
type Logic struct {
	ds              comm.DataService
	sender          comm.Sender
	localServ       Server
	others          []Server
	state           State
	tm              *time.Timer
	logRepTm        *time.Timer
	stopHeartbeatCh chan bool
	cmdCh           chan comm.Command
	closeCmdCh      chan bool
	closeLogRepCh   chan bool
	logEntries      []comm.Entry
}

type State struct {
	currentTerm int32
	votedFor    int32
	commitIndex int32
	lastApplied int32
	nextIndex   map[int32]int32
	matchIndex  map[int32]int32
}

const (
	Follower = iota
	Candidate
	Leader
)

var RoleStr = []string{"Follower", "Candidate", "Leader"}

const (
	TimeOut    = 1000
	LOW        = 2000
	HIGH       = 5000
	CMD_CH_LEN = 10
)

// create a logic instance
func New(l Server) *Logic {
	// nextIndex := make(map[int32]int32, len(o))
	// matchIndex := make(map[int32]int32, len(o))
	// for _, s := range o {
	// 	id, _ := s.GetId()
	// 	nextIndex[int32(id)] = int32(0)
	// 	matchIndex[int32(id)] = int32(0)
	// }
	log := &Logic{localServ: l,
		state:           State{nextIndex: make(map[int32]int32), matchIndex: make(map[int32]int32)},
		stopHeartbeatCh: make(chan bool),
		cmdCh:           make(chan comm.Command, CMD_CH_LEN),
		closeCmdCh:      make(chan bool),
		closeLogRepCh:   make(chan bool),
		logEntries:      make([]comm.Entry, 0, 0)}

	go log.recvCmd()
	return log
}

// subscribe services
func (l *Logic) Subscribe(c comm.DataService) {
	l.ds = c
}

// yeah! start the logic module.
func (l *Logic) Run() {
	glog.Info("I'm ", RoleStr[l.localServ.Role])
	l.tm = time.NewTimer(randomTime())
	// start the timer
	go func() {
		for {
			select {
			case <-l.tm.C:
				go l.electLeader()
				l.tm.Reset(randomTime())
			}
		}
	}()

	// waiting for the args from data service
	for {
		d := <-l.ds.GetDataChan()
		l.tm.Reset(randomTime())
		go l.argsHandler(d)
	}
}

func (l *Logic) Connect(s Server) error {
	if contains(l.others, s.Addr) {
		return errors.New("duplicate server")
	}
	l.others = append(l.others, s)
	id, _ := s.GetId()
	l.state.nextIndex[int32(id)] = int32(0)
	l.state.matchIndex[int32(id)] = int32(0)
	return nil
}

// handle
func (l *Logic) argsHandler(dc comm.DataChan) {
	select {
	case args := <-dc.Vc.Args:
		glog.Info("recv vote: ", args)
		if args.Term < l.state.currentTerm {
			glog.Info("ignore vote requst with term:", args.Term, " current term is ", l.state.currentTerm)
			dc.Vc.Result <- &comm.VoteResult{Term: args.Term, VoteGranted: false}
			return
		}

		if l.state.votedFor > 0 && args.Term == l.state.currentTerm {
			glog.Info("ignore vote requst with term:", args.Term, " has voted for ", l.state.votedFor)
			dc.Vc.Result <- &comm.VoteResult{Term: args.Term, VoteGranted: false}
			return
		}

		if args.Term > l.state.currentTerm {
			l.state.currentTerm = args.Term
			if l.localServ.Role == Leader {
				l.localServ.Role = Candidate
				l.stopHeartbeatCh <- true
			}
		}

		l.state.votedFor = args.CandidateId
		dc.Vc.Result <- &comm.VoteResult{Term: args.Term, VoteGranted: true}
	case args := <-dc.Ac.Args:
		if len(args.Entries) == 0 {
			// recv heartbeat, leader come up, change role to follower
			if args.Term > l.state.currentTerm {
				if l.state.currentTerm == Leader {
					l.closeLogRepCh <- true
				}
			}
			l.localServ.Role = Follower
			glog.Info("recv heart beat")
		} else {
			glog.Info("AppEntry:", args)
			// check term number
			if args.Term < l.state.currentTerm {
				glog.Info("app entry request's term is too old")
				dc.Ac.Result <- &comm.AppEntryResult{Term: l.state.currentTerm, Success: false}
				return
			}

			// check whether the prev log's term and prev term are matched
			if len(l.logEntries) <= int(args.PrevLogIndex) {
				// prev log index is greater than log entries' length.
				glog.Error("prev log index is out of boundary")
				dc.Ac.Result <- &comm.AppEntryResult{Term: l.state.currentTerm, Success: false}
				return
			} else {
				// check whether the prev log index is -1, which means the current is the first entry.
				if args.PrevLogIndex >= 0 {
					// prev terms are matched.
					if l.logEntries[args.PrevLogIndex].Term == args.PrevLogTerm {
						// check whether current log index's entry is exist in the server
						if len(l.logEntries) > int(args.PrevLogIndex+1) {
							// remove the current log entry and all that follow it
							l.logEntries = l.logEntries[:args.PrevLogIndex+1]
						}
					} else {
						// the term number of the prev log entry is not equal to the arg's prev term num.
						glog.Info("term are not matched")
						dc.Ac.Result <- &comm.AppEntryResult{Term: l.state.currentTerm, Success: false}
						return
					}
				}

				// append the entries to local server
				for _, e := range args.Entries {
					l.logEntries = append(l.logEntries, e)
				}

				if args.LeaderCommit > l.state.commitIndex {
					l.state.commitIndex = int32(min(int(args.LeaderCommit), len(l.logEntries)-1))
				}
				dc.Ac.Result <- &comm.AppEntryResult{Term: l.state.currentTerm, Success: true}
			}
		}
	}
}

func (l *Logic) electLeader() {
	l.state.currentTerm++
	glog.Info("Term:", l.state.currentTerm)
	l.localServ.Role = Candidate
	l.state.votedFor = 0
	glog.Info("I'm candidate, start to elect leader")

	// log.Println("Send vote Request")
	rltch := make(chan comm.VoteResult, len(l.others))
	cid, err := l.localServ.GetId()
	if err != nil {
		glog.Info("failed to get candidate id of ", l.localServ.Addr)
		return
	}

	rlts := make([]comm.VoteResult, 0, 0)
	// vote for self
	l.state.votedFor = int32(cid)
	rlts = append(rlts, comm.VoteResult{})

	args := comm.VoteArgs{Term: l.state.currentTerm, CandidateId: int32(cid)}
	for _, s := range l.others {
		go func(serv Server) {
			rlt, err := l.vote(serv.Addr, args, time.Duration(LOW))
			if err != nil {
				glog.Error("request vote of ", serv.Addr, " err:", err)
				return
			}
			rltch <- rlt
		}(s)
	}

	// wait the result
	for {
		select {
		case rlt := <-rltch:
			glog.Info("result vote:", rlt)
			if rlt.Term < l.state.currentTerm {
				// glog.Info("ignore the vote result")
				continue
			}
			if rlt.VoteGranted {
				rlts = append(rlts, rlt)
			}
			glog.Info("vote num:", len(rlts))
			if len(rlts) > (len(l.others) / 2) {
				l.localServ.Role = Leader
				glog.Info("I'm leader, vote num:", len(rlts), " term:", l.state.currentTerm)
				l.tm.Stop()
				// start to send heatbeat or do log replication
				// go l.heartBeat()
				go l.logReplication()
				go l.AppLogTest()
			} else {
				glog.Info("not enough vote:", len(rlts))
			}
		case <-time.After(LOW * time.Millisecond):
			return
		}
	}
}

func (l *Logic) vote(addr string, args comm.VoteArgs, tmout time.Duration) (comm.VoteResult, error) {
	ch := make(chan struct {
		rlt comm.VoteResult
		err error
	})
	go func() {
		rlt := comm.VoteResult{}
		// log.Println("VoteRequest ", addr)
		err := l.sender.RequestVote(addr, args, &rlt)
		ch <- struct {
			rlt comm.VoteResult
			err error
		}{rlt, err}
	}()

	for {
		select {
		case v := <-ch:
			return v.rlt, v.err
		case <-time.After(tmout * time.Millisecond):
			return comm.VoteResult{}, errors.New("time out")
		}
	}
}

func (l *Logic) appEntry(addr string, args comm.AppEntryArgs, tmout time.Duration) (comm.AppEntryResult, error) {
	ch := make(chan struct {
		rlt comm.AppEntryResult
		err error
	})
	go func() {
		rlt := comm.AppEntryResult{}
		err := l.sender.AppEntries(addr, args, &rlt)
		ch <- struct {
			rlt comm.AppEntryResult
			err error
		}{rlt, err}
	}()

	for {
		select {
		case v := <-ch:
			return v.rlt, v.err
		case <-time.After(tmout * time.Millisecond):
			return comm.AppEntryResult{}, errors.New("time out")
		}
	}
}

// through the cmd to log replication channel, the reason of using channel to
// recv the cmd is: this function can invoked concurrently.
func (l *Logic) ReplicateCmd(cmd comm.Command) {
	glog.Info(l.cmdCh)
	l.cmdCh <- cmd
}

func (l *Logic) logReplication() {
	l.logRepTm = time.NewTimer(1 * time.Millisecond)
	for {
		select {
		case <-l.logRepTm.C:
			glog.Info("logRepTimer")
			for _, serv := range l.others {
				leaderid, _ := l.localServ.GetId()
				go func(s Server) {
					id, _ := s.GetId()
					glog.Info("id:", id)
					nxtIndex := l.state.nextIndex[int32(id)]
					if 0 <= nxtIndex && nxtIndex < int32(len(l.logEntries)) {
						entry := l.logEntries[nxtIndex]
						prevTerm := int32(0)
						if nxtIndex <= 0 {
							prevTerm = -1
						} else {
							prevTerm = l.logEntries[nxtIndex-1].Term
						}
						// generate AppendEntryArg
						entries := make([]comm.Entry, 0, 0)
						entries = append(entries, entry)
						arg := comm.AppEntryArgs{
							Term:         l.state.currentTerm,
							LeaderId:     int32(leaderid),
							PrevLogIndex: nxtIndex - 1,
							PrevLogTerm:  prevTerm,
							Entries:      entries,
							LeaderCommit: l.state.commitIndex}
						rlt, err := l.appEntry(s.Addr, arg, time.Duration(LOW/2))
						if err != nil {
							glog.Error(err)
							return
						}
						if rlt.Success {
							glog.Info("log rep to ", s.Addr, " success")
							l.state.matchIndex[int32(id)] = nxtIndex
							l.state.nextIndex[int32(id)] = nxtIndex + 1
							glog.Info("move next index of ", s.Addr, " index:", l.state.nextIndex[int32(id)])
						} else {
							glog.Info("log rep to ", s.Addr, " failed")
							if rlt.Term > l.state.currentTerm {
								glog.Info("current term < ", s.Addr, "'s term'")
								return
							}
							l.state.nextIndex[int32(id)] = nxtIndex - 1
							glog.Info("set back index of ", s.Addr, " index:", l.state.nextIndex[int32(id)])
						}
					} else {
						// send heart beat, or maybe do nothing.
						arg := comm.AppEntryArgs{Term: l.state.currentTerm}
						glog.Info("Term:", l.state.currentTerm, " send heart beat to ", s.Addr)
						_, err := l.appEntry(s.Addr, arg, time.Duration(LOW/2))
						if err != nil {
							glog.Info("send hb to ", s.Addr, " failed, err:", err)
						}
					}
				}(serv)
			}
			l.logRepTm.Reset(time.Duration(LOW/2) * time.Millisecond)
		case <-l.closeLogRepCh:
			return
		}
	}
}

func (l *Logic) recvCmd() {
	for {
		select {
		case cmd := <-l.cmdCh:
			glog.Info("recv cmd:", cmd)
			// write the log to disk
			entry := comm.Entry{Cmd: cmd.Serialise(), Term: l.state.currentTerm}
			l.cmdToDisk(entry.Cmd)
			l.logEntries = append(l.logEntries, entry)
		case <-l.closeCmdCh:
			return
		}
	}
}

func (l *Logic) cmdToDisk(cmd string) {

}

func randomTime() time.Duration {
	t := random(LOW, HIGH)
	// glog.Info("rand time: ", t)
	return time.Duration(t) * time.Millisecond
}

func random(min, max int) int {
	return rand.Intn(max-min) + min
}

// Close the whole logic module
func (l *Logic) Close() {
	l.closeCmdCh <- true
	l.closeLogRepCh <- true
}

func min(a int, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}

func contains(others []Server, addr string) bool {
	for _, v := range others {
		if v.Addr == addr {
			return true
		}
	}
	return false
}

type command struct {
	cmd string
}

func (cmd command) Serialise() string {
	return cmd.cmd
}

func (cmd command) UnSerialise(s string) {
	cmd.cmd = s
}

func (l *Logic) AppLogTest() {
	i := 0
	for {
		cmd := command{"+" + strconv.Itoa(i)}
		i++
		l.ReplicateCmd(cmd)
		time.Sleep(time.Second)
	}
}

func (l Logic) GetMatchedNum() {

}
