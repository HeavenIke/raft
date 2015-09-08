package comm

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
	Term         int32
	CandidateId  int32
	LastLogIndex int32
	LastLogTerm  int32
}

type VoteResult struct {
	Term        int32
	VoteGranted bool
}

type AppEntryArgs struct {
	Term         int32
	LeaderId     int32
	PrevLogIndex int32
	PrevLogTerm  int32
	Entries      []Entry
	LeaderCommit int32
}

type AppEntryResult struct {
	Term    int32
	Success bool
}

type Entry struct {
	cmd int32
}

type DataChan struct {
	Vc VoteChan
	Ac AppEntryChan
}

type Service struct {
	DataChan
}

func NewService() *Service {
	s := &Service{}
	s.Vc = NewVoteChan()
	s.Ac = NewAppEntryChan()
	return s
}

func (dc DataChan) Close() {
	close(dc.Vc.Args)
	close(dc.Vc.Result)
	close(dc.Ac.Args)
	close(dc.Ac.Result)
}

func (r *Service) RequestVote(args *VoteArgs, result *VoteResult) error {
	r.Vc.Args <- args
	*result = *(<-r.Vc.Result)
	return nil
}

func (r *Service) AppendEntries(args *AppEntryArgs, result *AppEntryResult) error {
	r.Ac.Args <- args
	*result = *(<-r.Ac.Result)
	return nil
}
