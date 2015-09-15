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
	// if contains(r.others, addr) {
	// 	return errors.New("duplicate addr:" + addr)
	// }
	return r.logic.Connect(logic.Server{Addr: addr, Role: logic.Follower})
	// r.others = append(r.others, logic.Server{Addr: addr, Role: logic.Follower})
	// glog.Info("other len:", len(r.others))
	// glog.Info(r.others)
	// return nil
}

func (r *Raft) Run() {
	r.logic.Run()
}

func (r *Raft) ReplicateCmd(cmd comm.Command) {
	r.logic.ReplicateCmd(cmd)
}
