package main

import "github.com/heavenike/raft"

func main() {
	r := raft.New(":1234")
	r.Connect(":2345")
	r.Connect(":3456")
	r.Connect(":4567")
	r.Connect(":5678")
	r.Run()

	c := make(chan int)
	<-c
}
