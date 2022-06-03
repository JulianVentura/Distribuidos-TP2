package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type JoinerConfig struct {
	keep_id           bool // If true, id will be delivered. If not it will be filtered out
	base_body_size    uint // Number of columns of base imput body to preserve
	to_join_body_size uint // Number of columns of 'to join' imput body to preserve
}

type Joiner struct {
	table           map[string]string
	Parser          utils.MessageParser
	config          JoinerConfig
	post_arrival    uint
	comment_arrival uint
	join_written    uint
}

func NewJoiner(config JoinerConfig) Joiner {

	return Joiner{
		table:           make(map[string]string, 100), //Initial value
		Parser:          utils.NewParser(),
		config:          config,
		post_arrival:    0,
		comment_arrival: 0,
		join_written:    0,
	}
}

func (self *Joiner) info() {
	//TODO: Eliminate this. It's only for debugging purposes
	log.Infof("Posts: %v, Comments: %v, Written: %v", self.post_arrival, self.comment_arrival, self.join_written)
}

func (self *Joiner) Add(input string) {
	log.Debugf("Received: %v", input)
	self.post_arrival += 1
	splits := self.Parser.Read(input)
	if len(splits) < int(self.config.base_body_size)+1 {
		log.Errorf("Received bad base input on joiner")
		return
	}

	id := splits[0]
	body := ""
	if self.config.base_body_size > 0 {
		to_merge := splits[1 : self.config.base_body_size+1]
		body = self.Parser.Write(to_merge)
	}
	self.table[id] = body
}

func (self *Joiner) Join(input string) (string, error) {
	self.comment_arrival += 1
	splits := self.Parser.Read(input)
	if len(splits) < int(self.config.to_join_body_size)+1 {
		log.Errorf("Received bad join input on joiner")
		return "", fmt.Errorf("Received bad base input on joiner")
	}

	id := splits[0]
	body, exists := self.table[id]
	if !exists {
		return "", fmt.Errorf("entry do not exist")
	}

	result := make([]string, 0, 3)
	if self.config.keep_id {
		result = append(result, id)
	}
	if len(body) > 0 {
		result = append(result, body)
	}
	if self.config.to_join_body_size > 0 {
		to_merge := splits[1 : self.config.to_join_body_size+1]
		join_body := self.Parser.Write(to_merge)
		result = append(result, join_body)
	}

	self.join_written += 1

	return self.Parser.Write(result), nil
}

func test_function() {
	to_add := []string{
		"id1,0.23,a",
		"id2,0.1",
		"id3,0.23",
	}

	to_join := []string{
		"id1,z",
		"id1,b",
		"id5,0.5,a",
		"id2,0.23,a",
		"id3,-0.5,a,b",
	}

	joiner := NewJoiner(JoinerConfig{keep_id: false, base_body_size: 1, to_join_body_size: 0})
	joiner.Parser = utils.CustomParser(',')

	for _, line := range to_add {
		joiner.Add(line)
	}

	for _, line := range to_join {
		r, err := joiner.Join(line)
		if err != nil {
			fmt.Printf("%v: X\n", line)
		} else {
			fmt.Printf("%v\n", r)
		}
	}
}
