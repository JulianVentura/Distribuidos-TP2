package main

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostScoreAvgCalculator struct {
	Sum     int64
	Counter int64
}

func NewCalculator() PostScoreAvgCalculator {
	return PostScoreAvgCalculator{
		Sum:     0,
		Counter: 0,
	}
}

func (self *PostScoreAvgCalculator) add(input string) {
	log.Debugf("Received: %v", input)

	split := strings.Split(input, ",")
	score, _ := strconv.ParseInt(split[0], 10, 32)
	count, _ := strconv.ParseInt(split[1], 10, 32)

	self.Sum += score
	self.Counter += count
}

func (self *PostScoreAvgCalculator) get_result() string {
	return fmt.Sprintf("%.4f", float64(self.Sum)/float64(self.Counter))
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
