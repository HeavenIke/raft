package logic

import (
	"strconv"
	"testing"

	"github.com/iketheadore/raft/comm"
)

// func TestSendB(t *testing.T) {
// 	flag.Set("logtostderr", "true")
// 	flag.Parse()
// 	glog.Info("Testing")
// 	lserv := Server{Addr: ":1234", Role: Follower}
// 	others := []Server{Server{Addr: ":2345", Role: Follower}}
// 	l := New(lserv, others)
// 	for {
// 		l.sendHB()
// 		time.Sleep(time.Second)
// 	}
// }

func TestLoadLogEntry(t *testing.T) {
	logic := New(Server{Addr: ":1234", Role: Follower})

	for i := 0; i < 10; i++ {
		logic.entryToDisk(comm.Entry{Term: 1, LogIndex: i, Cmd: "+" + strconv.Itoa(i)})
	}

	for i := 2; i < 12; i++ {
		logic.entryToDisk(comm.Entry{Term: 2, LogIndex: i, Cmd: "+" + strconv.Itoa(i)})
	}

	logic.loadLogEntry("logentry.data")

	if logic.logEntries[0].Term != 1 {
		t.Fail()
	}

	if logic.logEntries[1].Term != 1 {
		t.Fail()
	}

	if logic.logEntries[2].Term != 2 {
		t.Fail()
	}
}
