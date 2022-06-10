package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type CommentDigestor struct {
	permalingRegex *regexp.Regexp
	sentimentRegex *regexp.Regexp
	bodyRegex      *regexp.Regexp
	Parser         utils.MessageParser
}

func NewDigestor() CommentDigestor {
	self := CommentDigestor{
		Parser: utils.NewParser(),
	}
	self.generateRegex()

	return self
}

func (self *CommentDigestor) filter(input string) (string, error) {
	log.Debugf("Received: %v", input)

	permalinkIdx := 6
	bodyIdx := 7
	sentimentIdx := 8

	splits := self.Parser.Read(input)
	if len(splits) != 10 {
		return "", fmt.Errorf("invalid input")
	}

	permalink := splits[permalinkIdx]
	body := splits[bodyIdx]
	sentiment := splits[sentimentIdx]

	ok := true
	ok = ok && self.sentimentRegex.MatchString(sentiment)
	ok = ok && self.bodyRegex.MatchString(body)
	ok = ok && (body != "[deleted]") && (body != "[removed]")
	if !ok {
		return "", fmt.Errorf("invalid input")
	}
	postId, err := self.getPostId(permalink)

	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	output := []string{postId, sentiment, body}
	result := self.Parser.Write(output)

	return result, nil
}

func (self *CommentDigestor) generateRegex() {
	sentimentRegex := `^[+-]?(1(\.0+)?|(0\.[0-9]+))$`
	bodyRegex := `(.*)?\S+(.*)?` // Allow any space
	permalingRegex := `https://old\.reddit\.com/r/((\bme_irl\b)|(\bmeirl\b))/comments/([^/]+)/.*`

	//Build the regex
	self.sentimentRegex = regexp.MustCompile(sentimentRegex)
	self.bodyRegex = regexp.MustCompile(bodyRegex)
	self.permalingRegex = regexp.MustCompile(permalingRegex)
}

func (self *CommentDigestor) getPostId(input string) (string, error) {
	split := self.permalingRegex.FindStringSubmatch(input)
	if len(split) < 2 {
		return "", fmt.Errorf("Could not get id from input")
	}

	return split[4], nil
}
