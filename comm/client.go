package comm

import (
	"net/rpc"
)

type Sender struct {
	Addr []string
}

func (s *Sender) RequestVote(addr string, args VoteArgs, result *VoteResult) error {
	err := rpcRequest(addr, "Service.RequestVote", args, result)
	return err
}

func (s *Sender) AppEntries(addr string, args AppEntryArgs, result *AppEntryResult) error {
	err := rpcRequest(addr, "Service.AppendEntries", args, result)
	return err
}

func rpcRequest(addr string, method string, args interface{}, result interface{}) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return err
	}
	err = client.Call(method, args, result)
	if err != nil {
		return err
	}
	return nil
}
