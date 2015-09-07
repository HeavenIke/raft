package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type PointInfo struct {
	LocalName string
	Points    map[string]string // key is name, value is addr
}

var Points map[string]string
var LocalName string

// locad info rom config.json
func init() {
	f, err := ioutil.ReadFile("config/config.json")
	if err != nil {
		log.Fatalln("failed to load info from config.json")
	}
	p := PointInfo{}
	err = json.Unmarshal(f, &p)
	if err != nil {
		log.Fatalln("unmarshal failed:", err)
	}
	// log.Println(p)
	Points = p.Points
	LocalName = p.LocalName
}
