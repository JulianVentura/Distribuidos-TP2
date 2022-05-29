package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type CommentDigestor struct {
	permalink_r      *regexp.Regexp
	sentiment_r      *regexp.Regexp
	body_r           *regexp.Regexp
	parser           utils.MessageParser
	received_counter uint //TODO: Delete this
	written_counter  uint
}

func (self *CommentDigestor) info() {
	//TODO: Eliminate this. It's only for debugging purposes
	log.Infof("Received: %v. Written: %v", self.received_counter, self.written_counter)
}

func NewDigestor() CommentDigestor {
	self := CommentDigestor{
		parser: utils.NewParser(10),
	}
	self.generate_regex()

	return self
}

func (self *CommentDigestor) SetParser(parser utils.MessageParser) {
	self.parser = parser
}

func (self *CommentDigestor) filter(input string) (string, error) {
	log.Debugf("Received: %v", input)
	self.received_counter += 1

	permalink_idx := 6
	body_idx := 7
	sentiment_idx := 8

	splits, err := self.parser.Read(input)
	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	permalink := splits[permalink_idx]
	body := splits[body_idx]
	sentiment := splits[sentiment_idx]

	ok := true
	ok = ok && self.sentiment_r.MatchString(sentiment)
	ok = ok && self.body_r.MatchString(body)
	ok = ok && (body != "[deleted]") && (body != "[removed]")
	if !ok {
		return "", fmt.Errorf("invalid input")
	}
	post_id, err := self.get_post_id(permalink)

	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	output := []string{post_id, sentiment, body}
	result := self.parser.Write(output)

	self.written_counter += 1

	return result, nil
}

func (self *CommentDigestor) generate_regex() {
	sentiment_r := `^[+-]?(1(\.0+)?|(0\.[0-9]+))$`
	body_r := `(.*)?\S+(.*)?` // Allow any space
	permalink_r := `https://old\.reddit\.com/r/((\bme_irl\b)|(\bmeirl\b))/comments/([^/]+)/.*`

	//Build the regex
	self.sentiment_r = regexp.MustCompile(sentiment_r)
	self.body_r = regexp.MustCompile(body_r)
	self.permalink_r = regexp.MustCompile(permalink_r)
}

func (self *CommentDigestor) get_post_id(input string) (string, error) {
	split := self.permalink_r.FindStringSubmatch(input)
	if len(split) < 2 {
		return "", fmt.Errorf("Could not get id from input")
	}

	return split[4], nil
}

func test_function() {
	lines := []string{
		//Correct
		"comment,cag9p9z,2vegg,me_irl,False,1370912170,https://old.reddit.com/r/me_irl/comments/1g2h1a/,According to elsewhere on Reddit: Peggle 2,0.0,2",
		//Correct
		"comment,cag9p9z,2vegg,me_irl,False,1370912170,https://old.reddit.com/r/me_irl/comments/1g2h1a/me_irl/cag9p9z/,According to elsewhere on Reddit: Peggle 2,0.0,2",
		//Correct
		"comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
		//Fewer values
		"cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
		//Bad body
		"comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,,0.4404,1",
		//Bad sentiment
		"comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,,1",
		//Bad permalink
		"comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.ret.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
	}

	digestor := NewDigestor()
	digestor.SetParser(utils.CustomParser(',', 10))

	for id, line := range lines {
		result, err := digestor.filter(line)
		if err != nil {
			fmt.Printf("Line %v: Invalid\n", id+1)
		} else {
			fmt.Printf("Line %v: Valid\n", id+1)
			fmt.Printf("- (%v)\n", result)
		}
	}
}
