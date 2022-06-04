package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type SentimentAvgCalculator struct {
	posts    map[string][]float64
	Parser   utils.MessageParser
	arrivals uint
}

func NewCalculator() SentimentAvgCalculator {

	return SentimentAvgCalculator{
		posts:  make(map[string][]float64, 100), //Initial value
		Parser: utils.NewParser(),
	}
}

func (self *SentimentAvgCalculator) info() {
	//TODO: Eliminate this. It's only for debugging purposes
	log.Infof("Arrivals: %v, Written: %v", self.arrivals, len(self.posts))
}

func (self *SentimentAvgCalculator) add(input string) {
	log.Debugf("Received: %v", input)
	self.arrivals += 1
	split := self.Parser.Read(input)
	if len(split) != 3 {
		log.Errorf("Received bad formated input on SentimentAvgCalculator")
		return
	}
	postId := split[0]
	sentiment, err := strconv.ParseFloat(split[1], 64)
	if err != nil {
		log.Errorf("Sentiment bad formated: %v", err)
		return
	}

	_, exists := self.posts[postId]
	if !exists {
		self.posts[postId] = []float64{sentiment, 1.0}
		return
	}
	self.posts[postId][0] += sentiment
	self.posts[postId][1] += 1.0
}

func (self *SentimentAvgCalculator) getResult() []string {
	result := make([]string, 0, len(self.posts))
	for post, values := range self.posts {
		toWrite := []string{fmt.Sprint(post), fmt.Sprintf("%.4f", values[0]/values[1])}
		result = append(result, self.Parser.Write(toWrite))
	}

	return result
}

func testFunction() {
	lines := []string{
		"id1,0.23,a",
		"id1,0.46,a",
		"id5,0.5,a",
		"id3,0.23,a",
		"id5,-0.5,a",
	}

	adder := NewCalculator()
	adder.Parser = utils.CustomParser(',')

	work := adder.add

	for _, line := range lines {
		work(line)
	}

	result := adder.getResult()

	for _, r := range result {
		fmt.Printf("%v\n", r)
	}
}
