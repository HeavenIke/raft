// This package maintains the transition of states for servers.
package logic

import (
	"errors"
	"math/rand"
	"time"

	"github.com/golang/glog"
	"github.com/iketheadore/raft/comm"
)

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
	appArgCh        chan comm.AppEntryArgs
	closeCmdCh      chan bool
	closeAppCh      chan bool
	logEntries      []comm.AppEntryArgs
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
	LOW        = 300
	HIGH       = 500
	CMD_CH_LEN = 10
)

// create a logic instance
func New(l Server, o []Server) *Logic {
	nextIndex := make(map[int32]int32, len(o))
	matchIndex := make(map[int32]int32, len(o))
	for _, s := range o {
		id, _ := s.GetId()
		nextIndex[int32(id)] = int32(0)
		matchIndex[int32(id)] = int32(0)
	}
	glog.Info(nextIndex)
	glog.Info(matchIndex)
	return &Logic{localServ: l,
		others:          o,
		state:           State{currentTerm: 0, votedFor: 0, commitIndex: 0, lastApplied: 0, nextIndex: nextIndex, matchIndex: matchIndex},
		stopHeartbeatCh: make(chan bool),
		cmdCh:           make(chan comm.Command, CMD_CH_LEN),
		appArgCh:        make(chan comm.AppEntryArgs),
		closeCmdCh:      make(chan bool),
		closeAppCh:      make(chan bool),
		logEntries:      make([]comm.AppEntryArgs, 0, 0)}
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

// handle
func (l *Logic) argsHandler(dc comm.DataChan) {
	select {
	case args := <-dc.Vc.Args:
		if args.Term < l.state.currentTerm {
			glog.Info("ignore vote requst with term:", args.Term, " current term is ", l.state.currentTerm)
			return
		}

		if l.state.votedFor > 0 && args.Term == l.state.currentTerm {
			glog.Info("ignore vote requst with term:", args.Term, " has voted for ", l.state.votedFor)
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
		dc.Vc.Result <- &comm.VoteResult{Term: args.Term}
	case args := <-dc.Ac.Args:
		glog.Info("App:", args)
		if args.Term == 0 {
			// recv heartbeat, leader come up, change role to follower
			l.localServ.Role = Follower
		}
		dc.Ac.Result <- &comm.AppEntryResult{}
	}
}

func (l *Logic) electLeader() {
	l.state.currentTerm++
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

	// vote for self
	// l.state.votedFor = int32(cid)

	args := comm.VoteArgs{Term: l.state.currentTerm, CandidateId: int32(cid)}
	for _, s := range l.others {
		go func(serv Server) {
			rlt, err := l.vote(serv.Addr, args, time.Duration(TimeOut))
			if err != nil {
				return
			}
			rltch <- rlt
		}(s)
	}

	// wait the result
	rlts := make([]comm.VoteResult, 0, 0)
	for {
		select {
		case rlt := <-rltch:
			glog.Info("vote:", rlt, " term:", l.state.currentTerm)
			if rlt.Term < l.state.currentTerm {
				// glog.Info("ignore the vote result")
				continue
			}
			rlts = append(rlts, rlt)
			glog.Info("vote num:", len(rlts))
			if len(rlts) > (len(l.others) / 2) {
				l.localServ.Role = Leader
				glog.Info("I'm leader, vote num:", len(rlts), " term:", l.state.currentTerm)
				l.tm.Stop()
				// start to send heatbeat or do log replication
				go l.heartBeat()
				go l.logReplication()
			} else {
				glog.Info("not enouth vote:", len(rlts))
			}
		case <-time.After(TimeOut * time.Millisecond):
			return
		}
	}
}

func (l *Logic) heartBeat() {
	glog.Info("start sending heartbeat")
	l.sendHB()
	for {
		select {
		case <-time.After(time.Duration(LOW/2) * time.Millisecond):
			l.sendHB()
		case <-l.stopHeartbeatCh:
			glog.Info("stop sending heartBeat")
			return
		}
	}
}

func (l *Logic) sendHB() {
	ch := make(chan comm.AppEntryResult, len(l.others))
	for _, serv := range l.others {
		go func(s Server) {
			arg := comm.AppEntryArgs{}
			glog.Info("send heart beat")
			rlt, err := l.appEntry(s.Addr, arg, time.Duration(LOW/2))
			if err != nil {
				glog.Info("send hb failed, err:", err)
				return
			}
			ch <- rlt
		}(serv)
	}

	go func() {
		rlts := make([]comm.AppEntryResult, 0, 0)
		for {
			select {
			case rlt := <-ch:
				rlts = append(rlts, rlt)
			case <-time.After(time.Duration(LOW/2) * time.Millisecond):
				if len(rlts) <= (len(l.others) / 2) {
					glog.Info("Not enough server in cluster, change role to candidate")
					l.localServ.Role = Candidate
					l.stopHeartbeatCh <- true
					return
				}
			}
		}
	}()
}

func (l *Logic) vote(addr string, args comm.VoteArgs, tmout time.Duration) (comm.VoteResult, error) {
	ch := make(chan comm.VoteResult)
	go func() {
		rlt := comm.VoteResult{}
		// log.Println("VoteRequest ", addr)
		err := l.sender.RequestVote(addr, args, &rlt)
		if err != nil {
			return
		}
		ch <- rlt
	}()

	for {
		select {
		case rlt := <-ch:
			return rlt, nil
		case <-time.After(tmout * time.Millisecond):
			return comm.VoteResult{}, errors.New("vote time out")
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
	// log the cmd to disk
	// s := cmd.Serialise()
	// l.cmdToDisk(s)
	// e := comm.Entry{Cmd: s}
	// l.localServ.Entries = append(l.localServ.Entries, e)
}

func (l *Logic) logReplication() {
	go l.appEntryHandler()
	// go func() {
	// 	// monitor the server's appentry channel
	// 	for _, s := range l.others {
	// 		go func() {
	// 			for {
	// 				select {
	// 				case <-s.AppEntryRltCh:
	// 					// send commit
	//
	// 				}
	// 			}
	// 		}()
	// 	}
	// }()
	for {
		select {
		case cmd := <-l.cmdCh:
			glog.Info("recv cmd:", cmd)
			// write the log to disk
			e := comm.Entry{Cmd: cmd.Serialise()}
			l.cmdToDisk(e.Cmd)

			id, err := l.localServ.GetId()
			if err != nil {
				glog.Error("GetId failed")
				continue
			}
			appArgs := comm.AppEntryArgs{
				Term:     l.state.currentTerm,
				LeaderId: int32(id)}

			appArgs.PrevLogIndex = int32(len(l.logEntries) - 2)
			appArgs.PrevLogTerm = l.logEntries[appArgs.PrevLogIndex].Term
			// l.logEntries = append(l.logEntries, appArgs)
			go func() {
				l.appArgCh <- appArgs
			}()
		case <-l.closeCmdCh:
			return
		}
	}
}

func (l *Logic) appEntryHandler() {
	for {
		select {
		case arg := <-l.appArgCh:
			l.logEntries = append(l.logEntries, arg)
		case <-l.logRepTm.C:
			// for _, s := range l.others {
			// id, _ := s.GetId()
			// glog.Info("id:", id)
			// glog.Info("nextIndex:", l.state.nextIndex)
			// logIndex := l.state.nextIndex[int32(id)]
			// if logIndex < int32(len(l.logEntries)) {
			// 	arg := l.logEntries[logIndex]
			// 	arg.Term = l.state.currentTerm
			// 	rlt, err := l.appEntry(s.Addr, arg, time.Duration(LOW/2))
			// 	if err != nil {
			// 		glog.Error(err)
			// 		continue
			// 	}
			// 	if rlt.Success {
			// 		l.state.matchIndex[int32(id)] = logIndex
			// 		l.state.nextIndex[int32(id)] = logIndex + 1
			// 	} else {
			// 		l.state.nextIndex[int32(id)] = logIndex - 1
			// 	}
			// } else {
			// 	// send heart beat, or maybe do nothing.
			// }
			// }
			l.logRepTm.Reset(time.Duration(LOW/2) * time.Millisecond)
		case <-l.closeAppCh:
			return
		}
	}
}

func (l *Logic) cmdToDisk(cmd string) {

}

func randomTime() time.Duration {
	return time.Duration(random(LOW, HIGH)) * time.Millisecond
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// Close the whole logic module
func (l *Logic) Close() {
	l.closeCmdCh <- true
}
