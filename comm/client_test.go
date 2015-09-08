package comm

import (
	"log"
	"testing"
	"time"
)

// type VoteArgs struct {
// 	Term         int32
// 	CandidateId  int32
// 	LastLogIndex int32
// 	LastLogTerm  int32
// }

func TestRequestVote(t *testing.T) {
	i := 0
	for {
		log.Println("num:", i)
		i++
		s := Sender{}
		args := VoteArgs{}
		rlt := VoteResult{}
		err := s.RequestVote(":1234", args, &rlt)
		if err != nil {
			log.Fatalln("request vote error:", err)
		}
		log.Println(rlt)
		time.Sleep(4 * time.Second)
	}
}
