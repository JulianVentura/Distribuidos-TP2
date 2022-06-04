package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type StudentCommentFilter struct {
	Parser       utils.MessageParser
	specialWords []string
}

func NewFilter() StudentCommentFilter {

	return StudentCommentFilter{
		Parser: utils.NewParser(),
		specialWords: []string{
			"university",
			"college",
			"student",
			"teacher",
			"professor",
		},
	}
}

func (self *StudentCommentFilter) filter(input string) (string, error) {
	// self.arrivals += 1
	split := self.Parser.Read(input)
	if len(split) != 3 {
		return "", fmt.Errorf("Received bad formated input on SentimentAvgCalculator")
	}
	postId := split[0]
	body := split[2]

	wordFound := false
	for _, word := range self.specialWords {
		if strings.Contains(body, word) {
			wordFound = true
			break
		}
	}

	if !wordFound {
		return "", fmt.Errorf("Body does not have a special word")
	}

	log.Debugf("Written: %v", postId)
	return postId, nil
}

func testFunction() {
	lines := []string{
		"id1,0.23,i went to that college and i learned a lot",
		"id1,0.46",
		"id5,0.5,i was a stuttdent in a different coliege. I didn't learn very much",
		"id3,0.23,Well; I'm a professor at Cambridge University so I'm better than you",
		"id6,-0.5,I haven't done anything; so...",
	}

	filter := NewFilter()
	filter.Parser = utils.CustomParser(',')
	results := make([]string, 0, 10)
	for _, line := range lines {
		result, err := filter.filter(line)
		if err == nil {
			results = append(results, result)
		}
	}

	for _, r := range results {
		fmt.Println(r)
	}
}
