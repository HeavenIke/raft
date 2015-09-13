package logic

import (
	"strconv"
	"strings"

	"github.com/iketheadore/raft/comm"
)

type Server struct {
	Addr     string
	Role     int8
	NxtIndex int32

	// private member
	appEntryArgsCh chan comm.AppEntryArgs
	closeCh        chan bool
}

func NewServer(addr string, role int8) *Server {
	s := Server{Addr: addr, Role: role, NxtIndex: 0}
	go func() {
		for {
			select {
			case arg := <-s.appEntryArgsCh:
				//TODO handle arg

			case <-s.closeCh:
				return
			}
		}
	}()
	return &s
}

func (s *Server) Close() {
	s.closeCh <- true
}

func (s Server) GetId() (int, error) {
	v := strings.SplitN(s.Addr, ":", 2)
	return strconv.Atoi(v[1])
}

func (s *Server) AppEntry(arg comm.AppEntryArgs) {
	s.appEntryArgsCh <- arg
}
