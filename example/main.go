package main

import (
	"flag"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/iketheadore/raft"
)

func main() {
	listen := flag.String("L", ":1234", "listen address")
	dst := flag.String("D", ":2345,:3456,:4567,:5678", "the others address")
	flag.Set("logtostderr", "true")
	flag.Parse()
	r := raft.New(*listen)
	d := strings.Split(*dst, ",")
	for _, v := range d {
		s := strings.Trim(v, " ")
		glog.Info("connect ", s)
		r.Connect(s)
	}
	go func() {
		r.Run()
	}()

	time.Sleep(3 * time.Second)
	// // for testing
	// go func() {
	// cmd1 := command{"aaaa"}
	// 	r.ReplicateCmd(cmd1)
	// 	cmd2 := command{"bbbb"}
	// 	r.ReplicateCmd(cmd2)
	// 	cmd3 := command{"cccc"}
	// 	r.ReplicateCmd(cmd3)
	// }()
	//
	c := make(chan bool)
	<-c
}
