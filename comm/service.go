package comm

import "encoding/json"

// type Listener struct {
// 	DataChan
// }
//
// func listen(addr string, lc Listener) {
// 	raft := &lc
// 	rpc.Register(raft)
// 	rpc.HandleHTTP()
// 	l, e := net.Listen("tcp", addr)
// 	if e != nil {
// 		log.Fatal("listern error:", e)
// 	}
// 	go http.Serve(l, nil)
// }

type VoteChan struct {
	Args   chan *VoteArgs
	Result chan *VoteResult
}

func NewDataChan() DataChan {
	return DataChan{Vc: NewVoteChan(), Ac: NewAppEntryChan()}
}

func NewVoteChan() VoteChan {
	vc := VoteChan{}
	vc.Args = make(chan *VoteArgs)
	vc.Result = make(chan *VoteResult)
	return vc
}

type AppEntryChan struct {
	Args   chan *AppEntryArgs
	Result chan *AppEntryResult
}

func NewAppEntryChan() AppEntryChan {
	ac := AppEntryChan{}
	ac.Args = make(chan *AppEntryArgs)
	ac.Result = make(chan *AppEntryResult)
	return ac
}

type VoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type VoteResult struct {
	Term        int
	VoteGranted bool
}

type AppEntryArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Entry
	LeaderCommit int
}

type AppEntryResult struct {
	Term    int
	Success bool
}

type Entry struct {
	Term     int
	LogIndex int
	Cmd      string
}

func (e Entry) Serialise() (string, error) {
	data, err := json.Marshal(&e)
	return string(data), err
}

func (e *Entry) UnSerialise(data string) error {
	return json.Unmarshal([]byte(data), e)
}

type Command interface {
	Serialise() string
	UnSerialise(cmd string)
}

type DataChan struct {
	Vc VoteChan
	Ac AppEntryChan
}

type Service struct {
	dcc chan DataChan
}

func NewService(num int) *Service {
	s := &Service{dcc: make(chan DataChan, num)}
	return s
}

func (s Service) GetDataChan() <-chan DataChan {
	return s.dcc
}

func (dc DataChan) Close() {
	close(dc.Vc.Args)
	close(dc.Vc.Result)
	close(dc.Ac.Args)
	close(dc.Ac.Result)
}

func (r *Service) RequestVote(args *VoteArgs, result *VoteResult) error {
	dc := NewDataChan()
	r.dcc <- dc
	dc.Vc.Args <- args
	*result = *(<-dc.Vc.Result)
	dc.Close()
	return nil
}

func (r *Service) AppendEntries(args *AppEntryArgs, result *AppEntryResult) error {
	dc := NewDataChan()
	r.dcc <- dc
	dc.Ac.Args <- args
	*result = *(<-dc.Ac.Result)
	dc.Close()
	return nil
}
