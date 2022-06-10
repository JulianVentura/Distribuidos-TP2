package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type PostScoreAdder struct {
	Counter int64
	Sum     int64
	Parser  utils.MessageParser
}

func NewCalculator() PostScoreAdder {
	return PostScoreAdder{
		Sum:     0,
		Counter: 0,
		Parser:  utils.NewParser(),
	}
}

func (self *PostScoreAdder) add(input string) {
	log.Debugf("Received: %v", input)
	splits := self.Parser.Read(input)
	if len(splits) != 3 {
		log.Errorf("Received bad formated input on PostScoreAdder")
		return
	}
	score, err := strconv.ParseInt(splits[2], 10, 32)
	if err != nil {
		log.Errorf("Received bad formated input on PostScoreAdder: %v", err)
		return
	}

	self.Counter += 1
	self.Sum += score
}

func (self *PostScoreAdder) getResult() string {
	result := []string{fmt.Sprint(self.Sum), fmt.Sprint(self.Counter)}
	return self.Parser.Write(result)
}
