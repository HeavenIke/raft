// This package is in charge of sending rpc request and handling the received request.
package comm

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type Data interface {
	Type() int8
	Serialise() interface{}
}

type DataChan struct {
	Vc VoteChan
	Ac AppEntryChan
}

type ListenService struct {
	DataChan
	Addr string
}

func NewListener(addr string) *ListenService {
	l := &ListenService{Addr: addr}
	return l
}

func (ls *ListenService) Run() {
	serv := NewService()
	ls.Ac = serv.Ac
	ls.Vc = serv.Vc
	rpc.Register(serv)
	rpc.HandleHTTP()
	log.Println("listen on", ls.Addr)
	l, e := net.Listen("tcp", ls.Addr)
	if e != nil {
		log.Fatal("listern error:", e)
	}
	go http.Serve(l, nil)
}
