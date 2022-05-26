package main

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostAboveAvgFilter struct {
	avg float64
}

func NewFilter(avg float64) PostAboveAvgFilter {
	return PostAboveAvgFilter{avg: avg}
}

func (self *PostAboveAvgFilter) work(input string) (string, error) {
	log.Debugf("Received: %v", input)

	split := strings.Split(input, ",")
	id := split[0]
	m_url := split[1]
	score, _ := strconv.ParseFloat(split[2], 10)

	if score >= self.avg {
		return fmt.Sprintf("%v,%v", id, m_url), nil
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
