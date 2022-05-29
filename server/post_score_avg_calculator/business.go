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
		Parser:  utils.NewParser(2),
	}
}

func (self *PostScoreAvgCalculator) add(input string) {
	log.Debugf("Received: %v", input)

	split, err := self.Parser.Read(input)
	if err != nil {
		log.Errorf("Received bad formated input on PostScoreAvgCalculator: %v", err)
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

func (self *PostScoreAvgCalculator) get_result() string {
	result := fmt.Sprintf("%.4f", float64(self.Sum)/float64(self.Counter))
	return self.Parser.Write([]string{result})
}

func test_function() {
	lines := []string{
		"123,3",
		"215,5",
		"23,1",
		"815,7",
		"145642,2228",
	}

	adder := NewCalculator()
	adder.Parser = utils.CustomParser(',', 2)

	work := adder.add

	for _, line := range lines {
		work(line)
	}

	if adder.get_result() == "65.4269" {
		fmt.Println("OK")
	} else {
		fmt.Printf("ERROR: %v", adder.get_result())
	}
}
