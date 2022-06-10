package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"strings"
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
		return "", nil
	}

	return postId, nil
}
