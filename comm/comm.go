// This package is in charge of sending rpc request and handling the received request.
package comm

import (
	"net"
	"net/http"
	"net/rpc"

	"github.com/golang/glog"
)

type DataService interface {
	GetDataChan() <-chan DataChan
}

type Listener struct {
	Addr string
	Serv *Service
}

const (
	serviceNum = 5
)

func NewListener(addr string) Listener {
	l := Listener{Addr: addr}
	l.Serv = NewService(serviceNum)
	return l
}

func (ls Listener) GetDataChan() <-chan DataChan {
	return ls.Serv.GetDataChan()
}

func (ls *Listener) Run() {
	rpc.Register(ls.Serv)
	rpc.HandleHTTP()
	glog.Info("listen on", ls.Addr)
	l, e := net.Listen("tcp", ls.Addr)
	if e != nil {
		glog.Fatal("listern error:", e)
	}
	go http.Serve(l, nil)
}
