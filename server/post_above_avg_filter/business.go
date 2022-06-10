package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strconv"
)

type PostAboveAvgFilter struct {
	avg    float64
	Parser utils.MessageParser
}

func NewFilter(avg float64) PostAboveAvgFilter {
	return PostAboveAvgFilter{
		avg:    avg,
		Parser: utils.NewParser(),
	}
}

func (self *PostAboveAvgFilter) work(input string) (string, error) {

	split := self.Parser.Read(input)
	if len(split) != 3 {
		return "", fmt.Errorf("Received bad formated input")
	}
	id := split[0]
	memeUrl := split[1]
	score, err := strconv.ParseFloat(split[2], 10)
	if err != nil {
		return "", fmt.Errorf("Couldn't parse score: %v", err)
	}

	if score >= self.avg {
		r := []string{id, memeUrl}
		return self.Parser.Write(r), nil
	} else {
		return "", nil
	}
}
