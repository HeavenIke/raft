package comm

import (
	"fmt"
	"net/rpc"
)

type Sender struct {
	// SendChan map[string]DataChan
}

func (s *Sender) RequestVote(addr string, args VoteArgs, result *VoteResult) error {
	return rpcRequest(addr, "Service.RequestVote", args, result)
}

func (s *Sender) AppendEntries(addr string, args AppEntryArgs, result *AppEntryResult) error {
	return rpcRequest(addr, "Service.AppendEntries", args, result)
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

	fmt.Println(result)
	return nil
}
