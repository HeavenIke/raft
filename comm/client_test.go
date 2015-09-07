package comm

import (
	"log"
	"testing"
)

// type VoteArgs struct {
// 	Term         int32
// 	CandidateId  int32
// 	LastLogIndex int32
// 	LastLogTerm  int32
// }

func TestRequestVote(t *testing.T) {
	log.Println("start")
	s := Sender{}
	args := VoteArgs{}
	rlt := VoteResult{}
	err := s.RequestVote(":1234", args, &rlt)
	if err != nil {
		log.Fatalln("request vote error:", err)
	}
	log.Println(rlt)
	log.Println("over")
}
