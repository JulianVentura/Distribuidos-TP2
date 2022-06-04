package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostDigestor struct {
	postIdRegex     *regexp.Regexp
	memeUrlRegex    *regexp.Regexp
	postScoreRegex  *regexp.Regexp
	Parser          utils.MessageParser
	receivedCounter uint
	writtenCounter  uint
}

func NewDigestor() PostDigestor {

	self := PostDigestor{
		Parser: utils.NewParser(),
	}
	self.generateRegex()

	return self
}

// func (self *PostDigestor) info() {
// 	//TODO: Eliminate this. It's only for debugging purposes
// 	log.Infof("Received: %v. Written: %v", self.received_counter, self.written_counter)
// }

func (self *PostDigestor) filter(input string) (string, error) {

	log.Debugf("Received: %v", input)

	self.receivedCounter += 1
	postIdIdx := 1
	postMemeIdx := 8
	postScoreIdx := 11

	splits := self.Parser.Read(input)
	if len(splits) != 12 {
		return "", fmt.Errorf("Received bad formated input")
	}

	postId := splits[postIdIdx]
	postMeme := splits[postMemeIdx]
	postScore := splits[postScoreIdx]

	ok := true
	ok = ok && self.postIdRegex.MatchString(postId)
	ok = ok && self.memeUrlRegex.MatchString(strings.ToLower(postMeme))
	ok = ok && self.postScoreRegex.MatchString(postScore)
	if !ok {
		return "", fmt.Errorf("invalid input")
	}
	output := []string{postId, postMeme, postScore}

	result := self.Parser.Write(output)

	return result, nil
}

func (self *PostDigestor) generateRegex() {
	//Post id
	postIdRegex := `^[aA-zZ0-9]+$`
	//Meme url
	validFormats := []string{
		"jpg",
		"png",
		"gifv",
		"gif",
		"jpeg",
	}
	for i, format := range validFormats {
		validFormats[i] = fmt.Sprintf("(%v)", format)
	}
	// meme_url_reg := `((https)|(http))?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	// meme_url_reg := `^((https)|(http))?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)\.` + `({valid_formats})` + `(\?.*)?$`
	baseUrl := `^((https)|(http))?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)\.`
	final := `(\?.*)?$`
	memeUrlRegex := fmt.Sprintf("%v(%v)%v", baseUrl, strings.Join(validFormats, "|"), final)

	//Score
	postScoreRegex := `^-?[0-9]+$` //Support negative

	//Build the regex
	self.postIdRegex = regexp.MustCompile(postIdRegex)
	self.memeUrlRegex = regexp.MustCompile(memeUrlRegex)
	self.postScoreRegex = regexp.MustCompile(postScoreRegex)
}

func testFunction() {
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
	digestor.Parser = utils.CustomParser(',')

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
