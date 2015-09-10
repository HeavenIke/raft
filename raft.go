package raft

import (
	"errors"

	"github.com/heavenike/raft/comm"
	"github.com/heavenike/raft/logic"
)

type Raft struct {
	localServ logic.Server
	others    []logic.Server
	listener  comm.Listener
	sender    comm.Sender
}

func New(addr string) Raft {
	return Raft{localServ: logic.Server{Addr: addr, Role: logic.Follower},
		listener: comm.NewListener(addr)}
}

func (r *Raft) Connect(addr string) error {
	if contains(r.others, addr) {
		return errors.New("duplicate addr:" + addr)
	}
	r.others = append(r.others, logic.Server{Addr: addr, Role: logic.Follower})
	return nil
}

func (r *Raft) Run() {
	r.listener.Run()
	l := logic.New(r.localServ, r.others)
	l.Subscribe(r.listener)
	l.Run()
}

func contains(others []logic.Server, addr string) bool {
	for _, v := range others {
		if v.Addr == addr {
			return true
		}
	}
	return false
}
