package main

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SentimentAvgCalculator struct {
	posts map[string][]float64
}

func NewCalculator() SentimentAvgCalculator {

	return SentimentAvgCalculator{
		posts: make(map[string][]float64, 100), //Initial value
	}
}

func (self *SentimentAvgCalculator) add(input string) {
	log.Debugf("Received: %v", input)

	split := strings.Split(input, ",")
	p_id := split[0]
	sentiment, _ := strconv.ParseFloat(split[1], 64)

	_, exists := self.posts[p_id]
	if !exists {
		self.posts[p_id] = []float64{sentiment, 1.0}
		return
	}
	self.posts[p_id][0] += sentiment
	self.posts[p_id][1] += 1.0
}

func (self *SentimentAvgCalculator) get_result() []string {
	result := make([]string, 0, len(self.posts))
	for post, values := range self.posts {
		result = append(result, fmt.Sprintf("%v,%.4f", post, values[0]/values[1]))
	}

	return result
}

func test_function() {
	lines := []string{
		"id1,0.23,a",
		"id1,0.46,a",
		"id5,0.5,a",
		"id3,0.23,a",
		"id5,-0.5,a",
	}

	adder := NewCalculator()

	work := adder.add

	for _, line := range lines {
		work(line)
	}

	result := adder.get_result()

	for _, r := range result {
		fmt.Printf("%v\n", r)
	}
}
