package logic

import (
	"log"
	"time"

	"github.com/heavenike/raft/comm"
)

// subscription interface
type subscription interface {
	DataChannel() comm.DataChan
	Close() error
	Loop()
}

type Sub struct {
	datachannel comm.DataChan
	closing     chan chan error
}

func (s Sub) DataChannel() comm.DataChan {
	return s.datachannel
}

// The received data will be processed in this loop.
func (s Sub) Loop() {
	for {
		select {
		case args := <-s.DataChannel().Vc.Args:
			log.Println("VoteRequest:", args)
			s.DataChannel().Vc.Result <- &comm.VoteResult{}
		case args := <-s.DataChannel().Ac.Args:
			log.Println("AppEntries:", args)
			s.DataChannel().Ac.Result <- &comm.AppEntryResult{}
		// case rlt := <-s.DataChannel().Vc.Result:
		// 	log.Println("Vote Result:", rlt)
		// case rlt := <-s.DataChannel().Ac.Result:
		// 	log.Println("AppEntry Result:", rlt)
		case err := <-s.closing:
			log.Println("loop break")
			err <- nil
			return
		case <-time.After(5 * time.Second):
			log.Println("Time out")
		}
	}
}

func (s Sub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	err := <-errc
	s.datachannel.Close()
	return err
}
