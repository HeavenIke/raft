package logic

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/iketheadore/raft/comm"
)

type Server struct {
	Addr          string
	Role          int8
	NxtIndex      int32
	AppEntryRltCh chan comm.AppEntryResult

	// private member
	sender         comm.Sender
	appEntryArgsCh chan comm.AppEntryArgs
	appEntryArgs   []comm.AppEntryArgs
	closeCh        chan bool
}

const (
	APPARG_BUFF_SIZE = 10
	APPENTRY_ERR     = "time out"
)

func NewServer(addr string, role int8) *Server {
	s := Server{Addr: addr,
		Role:           role,
		NxtIndex:       0,
		AppEntryRltCh:  make(chan comm.AppEntryResult),
		appEntryArgsCh: make(chan comm.AppEntryArgs, APPARG_BUFF_SIZE),
		closeCh:        make(chan bool)}
	go func() {
		for {
			select {
			case arg := <-s.appEntryArgsCh:
				//TODO handle arg
				s.appEntryArgs = append(s.appEntryArgs, arg)
				s.appEntryHandler(arg)
			case <-s.closeCh:
				return
			}
		}
	}()
	return &s
}

func (s *Server) appEntryHandler(arg comm.AppEntryArgs) {
	// send AppendEntries to the specific server
	rlt, err := s.appEntry(arg, time.Duration(LOW/2))
	if err != nil {
		if err.Error() == APPENTRY_ERR {
			s.appEntryHandler(arg)
		} else {
			// retry after specific time out
			time.AfterFunc(time.Duration(LOW/2), func() {
				glog.Info("err:", err, " retry AppEntries")
				s.appEntryHandler(arg)
			})
		}
	}
	if rlt.Success {
		s.NxtIndex++
		s.AppEntryRltCh <- rlt
	} else {
		if (arg.PrevLogIndex) > 0 {
			s.appEntryHandler(s.appEntryArgs[arg.PrevLogIndex-1])
		} else {
			glog.Error("the first log entry must be success")
		}
	}
}

func (s Server) appEntry(args comm.AppEntryArgs, tmout time.Duration) (comm.AppEntryResult, error) {
	ch := make(chan struct {
		rlt comm.AppEntryResult
		err error
	})
	go func() {
		rlt := comm.AppEntryResult{}
		err := s.sender.AppEntries(s.Addr, args, &rlt)
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
			return comm.AppEntryResult{}, errors.New(APPENTRY_ERR)
		}
	}
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
