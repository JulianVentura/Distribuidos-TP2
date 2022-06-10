package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type JoinerConfig struct {
	keepId           bool // If true, id will be delivered. If not it will be filtered out
	baseBodySize     uint // Number of columns of base imput body to preserve
	toJoinBodySize   uint // Number of columns of 'to join' imput body to preserve
	filterDuplicates bool // Filter out duplicates before write
}

type Joiner struct {
	table  map[string]string
	joined map[string]bool
	Parser utils.MessageParser
	config JoinerConfig
}

func NewJoiner(config JoinerConfig) Joiner {

	return Joiner{
		table:  make(map[string]string, 100), //Initial value
		joined: make(map[string]bool, 100),
		Parser: utils.NewParser(),
		config: config,
	}
}

func (self *Joiner) Add(input string) {
	log.Debugf("Add Received: %v", input)
	splits := self.Parser.Read(input)
	if len(splits) < int(self.config.baseBodySize)+1 {
		log.Errorf("Received bad base input on joiner")
		return
	}

	id := splits[0]
	body := ""
	if self.config.baseBodySize > 0 {
		toMerge := splits[1 : self.config.baseBodySize+1]
		body = self.Parser.Write(toMerge)
	}
	self.table[id] = body
}

func (self *Joiner) Join(input string) (string, error) {
	log.Debugf("Join Received: %v", input)
	splits := self.Parser.Read(input)
	if len(splits) < int(self.config.toJoinBodySize)+1 {
		log.Errorf("Received bad join input on joiner")
		return "", fmt.Errorf("Received bad base input on joiner")
	}

	id := splits[0]
	body, exists := self.table[id]
	if !exists {
		return "", nil
	}

	result := make([]string, 0, 3)
	if self.config.keepId {
		result = append(result, id)
	}
	if len(body) > 0 {
		result = append(result, body)
	}
	if self.config.toJoinBodySize > 0 {
		toMerge := splits[1 : self.config.toJoinBodySize+1]
		joinBody := self.Parser.Write(toMerge)
		result = append(result, joinBody)
	}

	r := self.Parser.Write(result)

	if self.config.filterDuplicates {
		_, exists := self.joined[r]
		if exists {
			return "", nil
		} else {
			self.joined[r] = true
		}
	}

	return r, nil
}
