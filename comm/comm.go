// This package is in charge of sending rpc request and handling the received request.
package comm

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type DataService interface {
	GetDataChan() DataChan
}

type Listener struct {
	Addr string
	Serv *Service
}

func NewListener(addr string) Listener {
	l := Listener{Addr: addr}
	l.Serv = NewService()
	return l
}

func (ls Listener) GetDataChan() DataChan {
	return DataChan{Vc: ls.Serv.Vc, Ac: ls.Serv.Ac}
}

func (ls *Listener) Run() {
	rpc.Register(ls.Serv)
	rpc.HandleHTTP()
	log.Println("listen on", ls.Addr)
	l, e := net.Listen("tcp", ls.Addr)
	if e != nil {
		log.Fatal("listern error:", e)
	}
	go http.Serve(l, nil)
}
