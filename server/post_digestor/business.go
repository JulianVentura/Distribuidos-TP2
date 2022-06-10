package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type PostDigestor struct {
	postIdRegex    *regexp.Regexp
	memeUrlRegex   *regexp.Regexp
	postScoreRegex *regexp.Regexp
	Parser         utils.MessageParser
}

func NewDigestor() PostDigestor {

	self := PostDigestor{
		Parser: utils.NewParser(),
	}
	self.generateRegex()

	return self
}

func (self *PostDigestor) filter(input string) (string, error) {

	log.Debugf("Received: %v", input)

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
