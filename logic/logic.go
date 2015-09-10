// This package maintains the transition of states for servers.
package logic

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/heavenike/raft/comm"
)

// logic control module
type Logic struct {
	ds        comm.DataService
	sender    comm.Sender
	localServ Server
	others    []Server
	state     State
	cancelHB  chan int8
	tmstop    chan bool
	tm        *time.Timer
}

type State struct {
	currentTerm int32
	votedFor    int32
}

const (
	Follower = iota
	Candidate
	Leader
)

var RoleStr = []string{"Follower", "Candidate", "Leader"}

const (
	HB_STOP = 0
	TimeOut = 1000
	LOW     = 300
	HIGH    = 500
)

type Server struct {
	Addr string
	Role int8
}

// create a logic instance
func New(l Server, o []Server) *Logic {
	return &Logic{localServ: l,
		others:   o,
		state:    State{currentTerm: 0, votedFor: 0},
		cancelHB: make(chan int8),
		tmstop:   make(chan bool)}
}

func (s Server) GetCandidateId() (int, error) {
	v := strings.SplitN(s.Addr, ":", 2)
	return strconv.Atoi(v[1])
}

// subscribe services
func (l *Logic) Subscribe(c comm.DataService) {
	l.ds = c
}

// yeah! start the logic module.
func (l *Logic) Run() {
	glog.Info("I'm ", RoleStr[l.localServ.Role])
	l.tm = time.NewTimer(randomTime())
	go l.StartTimer()

	for {
		d := <-l.ds.GetDataChan()
		l.tm.Reset(randomTime())
		go l.argsHandler(d)
	}
}

func (l *Logic) StartTimer() {
	for {
		select {
		case <-l.tm.C:
			go l.electLeader()
			l.tm.Reset(randomTime())
		}
	}
}

func (l *Logic) argsHandler(dc comm.DataChan) {
	select {
	case args := <-dc.Vc.Args:
		// glog.Info("Recv Vote request:", args)
		if args.Term < l.state.currentTerm {
			// glog.Info("ignore vote requst with term:", args.Term, " current term is ", l.state.currentTerm)
			return
		}

		if l.state.votedFor > 0 && args.Term == l.state.currentTerm {
			// glog.Info("ignore vote requst with term:", args.Term, " has voted for ", l.state.votedFor)
			return
		}

		if args.Term > l.state.currentTerm {
			l.state.currentTerm = args.Term
			if l.localServ.Role == Leader {
				l.localServ.Role = Candidate
				l.cancelHB <- HB_STOP
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

// Close the whole logic module
func (l *Logic) Close() {
	// err := l.sub.Close()
	// if err != nil {
	// 	glog.Info("Close error:", err)
	// }
}

func (l *Logic) electLeader() {
	l.state.currentTerm++
	l.localServ.Role = Candidate
	l.state.votedFor = 0
	glog.Info("I'm candidate, start to elect leader")
	// log.Println("Send vote Request")
	rltch := make(chan comm.VoteResult, len(l.others))
	cid, err := l.localServ.GetCandidateId()
	if err != nil {
		glog.Info("failed to get candidate id of ", l.localServ.Addr)
		return
	}
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
				// start to send heatbeat to others
				go l.heartBeat()
			} else {
				// glog.Info("not enouth vote:", len(rlts))
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
		case <-l.cancelHB:
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
					l.cancelHB <- HB_STOP
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
			return comm.AppEntryResult{}, errors.New("AppEntry time out")
		}
	}
}

func randomTime() time.Duration {
	return time.Duration(random(LOW, HIGH)) * time.Millisecond
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
