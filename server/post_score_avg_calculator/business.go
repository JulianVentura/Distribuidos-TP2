package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type PostScoreAvgCalculator struct {
	Sum     int64
	Counter int64
	Parser  utils.MessageParser
}

func NewCalculator() PostScoreAvgCalculator {
	return PostScoreAvgCalculator{
		Sum:     0,
		Counter: 0,
		Parser:  utils.NewParser(),
	}
}

func (self *PostScoreAvgCalculator) add(input string) {
	log.Debugf("Received: %v", input)

	split := self.Parser.Read(input)
	if len(split) != 2 {
		log.Errorf("Received bad formated input on PostScoreAvgCalculator")
		return
	}
	score, err := strconv.ParseInt(split[0], 10, 32)
	if err != nil {
		log.Errorf("Received bad formated input on PostScoreAvgCalculator: %v", err)
		return
	}
	count, err := strconv.ParseInt(split[1], 10, 32)
	if err != nil {
		log.Errorf("Received bad formated input on PostScoreAvgCalculator: %v", err)
		return
	}

	self.Sum += score
	self.Counter += count
}

func (self *PostScoreAvgCalculator) getResult() string {
	result := fmt.Sprintf("%.4f", float64(self.Sum)/float64(self.Counter))
	return self.Parser.Write([]string{result})
}
