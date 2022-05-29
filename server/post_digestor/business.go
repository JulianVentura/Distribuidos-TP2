package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostDigestor struct {
	post_id_r        *regexp.Regexp
	meme_url_r       *regexp.Regexp
	post_score_r     *regexp.Regexp
	parser           utils.MessageParser
	received_counter uint
	written_counter  uint
}

func NewDigestor() PostDigestor {

	self := PostDigestor{
		parser: utils.NewParser(12),
	}
	self.generate_regex()

	return self
}

func (self *PostDigestor) SetParser(parser utils.MessageParser) {
	self.parser = parser
}

func (self *PostDigestor) info() {
	//TODO: Eliminate this. It's only for debugging purposes
	log.Infof("Received: %v. Written: %v", self.received_counter, self.written_counter)
}

func (self *PostDigestor) filter(input string) (string, error) {

	log.Debugf("Received: %v", input)

	self.received_counter += 1
	post_id_idx := 1
	post_meme_idx := 8
	post_score_idx := 11

	splits, err := self.parser.Read(input)
	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	post_id := splits[post_id_idx]
	post_meme := strings.ToLower(splits[post_meme_idx])
	post_score := splits[post_score_idx]

	ok := true
	ok = ok && self.post_id_r.MatchString(post_id)
	ok = ok && self.meme_url_r.MatchString(post_meme)
	if !ok {
		return "", fmt.Errorf("invalid input")
	}
	ok = ok && self.post_score_r.MatchString(post_score)

	output := []string{post_id, post_meme, post_score}

	result := self.parser.Write(output)

	return result, nil
}

func (self *PostDigestor) generate_regex() {
	//Post id
	post_id_reg := `^[aA-zZ0-9]+$`
	//Meme url
	meme_url_reg := `((https)|(http))?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	//Score
	post_score_reg := `^-?[0-9]+$` //Support negative

	//Build the regex
	self.post_id_r = regexp.MustCompile(post_id_reg)
	self.meme_url_r = regexp.MustCompile(meme_url_reg)
	self.post_score_r = regexp.MustCompile(post_score_reg)
}

func test_function() {
	lines := []string{
		//Correct
		"post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Fewer values
		"post,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Bad score
		"post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,word",
		//Empty Score
		"post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,",
		//Empty id
		"post,,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Bad id
		"post,@#$%@,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Empty meme url
		"post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,,,me irl,10",
	}

	digestor := NewDigestor()
	digestor.SetParser(utils.CustomParser(',', 12))

	for id, line := range lines {
		result, err := digestor.filter(line)
		if err != nil {
			fmt.Printf("Line %v: Invalid\n", id+1)
		} else {
			fmt.Printf("Line %v: Valid\n", id+1)
			fmt.Printf("%v\n", result)
		}
	}
}
