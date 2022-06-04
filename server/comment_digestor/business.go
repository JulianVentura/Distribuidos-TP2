package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type CommentDigestor struct {
	permalingRegex  *regexp.Regexp
	sentimentRegex  *regexp.Regexp
	bodyRegex       *regexp.Regexp
	Parser          utils.MessageParser
	receivedCounter uint //TODO: Delete this
	writtenCounter  uint
}

// func (self *CommentDigestor) info() {
// 	//TODO: Eliminate this. It's only for debugging purposes
// 	log.Infof("Received: %v. Written: %v", self.received_counter, self.written_counter)
// }

func NewDigestor() CommentDigestor {
	self := CommentDigestor{
		Parser: utils.NewParser(),
	}
	self.generateRegex()

	return self
}

func (self *CommentDigestor) filter(input string) (string, error) {
	log.Debugf("Received: %v", input)
	self.receivedCounter += 1

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

	self.writtenCounter += 1

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

func testFunction() {
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
	digestor.Parser = utils.CustomParser(',')

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
