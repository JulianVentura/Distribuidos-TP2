package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type PostAboveAvgFilter struct {
	avg    float64
	Parser utils.MessageParser
}

func NewFilter(avg float64) PostAboveAvgFilter {
	return PostAboveAvgFilter{
		avg:    avg,
		Parser: utils.NewParser(3),
	}
}

func (self *PostAboveAvgFilter) work(input string) (string, error) {
	log.Debugf("Received: %v", input)

	split, err := self.Parser.Read(input)
	if err != nil {
		log.Errorf("Received bad formated input: %v", err)
		return "", nil
	}
	id := split[0]
	m_url := split[1]
	score, err := strconv.ParseFloat(split[2], 10)
	if err != nil {
		log.Errorf("Couldn't parse score: %v", err)
		return "", nil
	}

	if score >= self.avg {
		r := []string{id, m_url}
		return self.Parser.Write(r), nil
	} else {
		return "", fmt.Errorf("Data under average")
	}
}

func test_function() {
	avg := float64(60)
	lines := []string{
		"a,url_a,41",
		"b,b,43",
		"c,b,23",
		"d,b,116",
		"e,b,75",
	}

	adder := NewFilter(avg)
	adder.Parser = utils.CustomParser(',', 3)

	work := adder.work

	for _, line := range lines {
		v, e := work(line)
		if e != nil {
			fmt.Printf("%v: Invalid\n", line)
		} else {
			fmt.Printf("%v: %v\n", line, v)
		}
	}
}
