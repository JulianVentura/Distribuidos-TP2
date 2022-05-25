package main

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostScoreAdder struct {
	Counter int64
	Sum     int64
}

func (self *PostScoreAdder) add(input string) {
	log.Debugf("Received: %v", input)

	score, _ := strconv.ParseInt(strings.Split(input, ",")[2], 10, 32)

	self.Counter += 1
	self.Sum += score
}

func (self *PostScoreAdder) get_result() string {
	return fmt.Sprintf("%v,%v", self.Sum, self.Counter)
}

func test_function() {
	lines := []string{
		"123,http://hola.png,14",
		"123,http://hola.png,-2",
		"123,http://hola.png,30",
		"123,http://hola.png,150",
	}

	adder := PostScoreAdder{}

	work := adder.add

	for _, line := range lines {
		work(line)
	}

	if adder.get_result() == "192,4" {
		fmt.Println("OK")
	} else {
		fmt.Println("ERROR")
	}
}
