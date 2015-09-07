package raft

import (
	"errors"

	"github.com/heavenike/raft/comm"
	"github.com/heavenike/raft/logic"
)

const (
	Follower = iota
	Candidate
	Leader
)

type Raft struct {
	localServ Server
	others    []Server
	listener  *comm.ListenService
}

type Server struct {
	Addr string
	St   int8
}

func New(addr string) Raft {
	return Raft{localServ: Server{Addr: addr, St: Follower}, listener: comm.NewListener(addr)}
}

func (r *Raft) Connect(addr string) error {
	if contains(r.others, addr) {
		return errors.New("duplicate addr:" + addr)
	}
	r.others = append(r.others, Server{Addr: addr, St: Follower})
	return nil
}

func (r *Raft) Run() {
	r.listener.Run()
	l := logic.New()
	l.BindListenChan(r.listener)
}

func contains(others []Server, addr string) bool {
	for _, v := range others {
		if v.Addr == addr {
			return true
		}
	}
	return false
}
