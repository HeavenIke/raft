package raft

import (
	"github.com/iketheadore/raft/comm"
	"github.com/iketheadore/raft/logic"
)

type Raft struct {
	localServ logic.Server
	listener  comm.Listener
	sender    comm.Sender
	logic     *logic.Logic
}

func New(addr string) *Raft {
	r := &Raft{localServ: logic.Server{Addr: addr, Role: logic.Follower}, listener: comm.NewListener(addr)}
	r.listener.Run()
	r.logic = logic.New(r.localServ)
	r.logic.Subscribe(r.listener)
	return r
}

func (r *Raft) Connect(addr string) error {
	return r.logic.Connect(logic.Server{Addr: addr, Role: logic.Follower})
}

func (r *Raft) Run() {
	r.logic.Run()
}

func (r *Raft) ReplicateCmd(cmd comm.Command) {
	r.logic.ReplicateCmd(cmd)
}
