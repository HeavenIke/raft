// This package maintains the transition of states for servers.
package logic

import (
	"log"

	"github.com/heavenike/raft/comm"
)

// logic control module
type Logic struct {
	sub    subscription
	sender comm.Sender
}

// create a logic instance
func New() *Logic {
	return &Logic{}
}

// subscribe services
func (l *Logic) Subscribe(c comm.DataService) {
	dc := c.GetDataChan()
	l.sub = Sub{datachannel: dc, closing: make(chan chan error)}
}

// yeah! start the logic module.
func (l *Logic) Run() {
	// start to listen loop
	go l.sub.Loop()
	go l.electLeader()
}

func (l *Logic) RequestVote(addr string, args comm.VoteArgs) error {
	rlt := comm.VoteResult{}
	if err := l.sender.RequestVote(addr, args, &rlt); err != nil {
		return err
	}
	return nil
}

func (l *Logic) AppEntryRequest(addr string, args comm.AppEntryArgs) error {
	rlt := comm.AppEntryResult{}
	if err := l.sender.AppEntries(addr, args, &rlt); err != nil {
		return err
	}
	return nil
}

// Close the whole logic module
func (l *Logic) Close() {
	err := l.sub.Close()
	if err != nil {
		log.Println("Close error:", err)
	}
}

func (l *Logic) electLeader() {

}
