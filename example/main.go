package main

import (
	"flag"
	"strings"

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
	r.Run()
}
