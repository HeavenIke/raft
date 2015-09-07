// This package maintains the transition of states for servers.
package logic

import (
	"log"

	"github.com/heavenike/raft/comm"
)

type Logic struct {
}

func New() Logic {
	return Logic{}
}

func (l *Logic) BindListenChan(c *comm.ListenService) {
	go func() {
		for {
			select {
			case args := <-c.Vc.Args:
				// TODO
				log.Println("VoteRequest:", args)
				c.Vc.Result <- &comm.VoteResult{}
			case args := <-c.Ac.Args:
				// TODO
				log.Println("AppEnties:", args)
			}
		}
	}()
}
