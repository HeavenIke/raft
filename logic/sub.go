package logic

// import (
// 	"github.com/golang/glog"
// 	"github.com/heavenike/raft/comm"
// )
//
// // subscription interface
// type subscription interface {
// 	DataChannel() []comm.DataChan
// 	Close() error
// 	Loop()
// }
//
// type Sub struct {
// 	inDatachannel  []comm.DataChan
// 	outDatachannel []comm.DataChan
// 	closing        chan chan error
// }
//
// func NewSub(dc []comm.DataChan) Sub {
// 	return Sub{
// 		inDatachannel:  dc,
// 		outDatachannel: comm.NewDataChan(),
// 		closing:        make(chan chan error)}
// }
//
// func (s Sub) DataChannel() []comm.DataChan {
// 	return s.outDatachannel
// }
//
// // The received data will be processed in this loop.
// func (s Sub) Loop() {
// 	for i, ch := range s.inDatachannel {
// 		select {
// 		case args := <-ch.Vc.Args:
// 			s.outDatachannel[i].Vc.Args <- args
// 		case args := <-ch.Ac.Args:
// 			s.outDatachannel[i].Ac.Args <- args
// 		case rlt := <-ch.Vc.Result:
// 			s.inDatachannel[i].Vc.Result <- rlt
// 		case rlt := <-ch.Ac.Result:
// 			s.inDatachannel[i].Ac.Result <- rlt
// 		case err := <-s.closing:
// 			glog.Info("loop break")
// 			err <- nil
// 			return
// 		}
// 	}
// 	glog.Info("exit loop")
// }
//
// func (s Sub) Close() error {
// 	errc := make(chan error)
// 	s.closing <- errc
// 	err := <-errc
// 	for i := 0; i < len(s.inDataChannel); i++ {
// 		s.inDatachannel[i].Close()
// 	}
// 	return err
// }
