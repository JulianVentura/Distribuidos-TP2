package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"math"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type BestSentimentAvgDownloader struct {
	Best_score float64
	Best       string
	Parser     utils.MessageParser
	arrivals   uint
}

func NewDownloader() BestSentimentAvgDownloader {
	return BestSentimentAvgDownloader{
		Best_score: math.Inf(-1),
		Best:       "",
		Parser:     utils.NewParser(),
	}
}

func (self *BestSentimentAvgDownloader) info() {
	//TODO: Eliminate this. It's only for debugging purposes
	log.Infof("Arrivals: %v", self.arrivals)
}

func (self *BestSentimentAvgDownloader) work(input string) {
	log.Debugf("Received: %v", input)
	self.arrivals += 1

	split := self.Parser.Read(input)
	if len(split) != 4 {
		log.Errorf("Received bad formated input")
		return
	}
	score, err := strconv.ParseFloat(split[3], 64)
	if err != nil {
		log.Errorf("Couldn't parse score: %v", err)
		return
	}

	if score > self.Best_score {
		self.Best_score = score
		self.Best = split[1]
	}
}

func (self *BestSentimentAvgDownloader) get_result() string {
	return self.Parser.Write([]string{self.Best})
}

func test_function() {
	lines := []string{
		"a,meme_url_1,2,0.23",
		"b,meme_url_2,2,0.0",
		"c,meme_url_3,2,0.87",
		"d,meme_url_4,2,-0.89",
	}

	downloader := NewDownloader()
	downloader.Parser = utils.CustomParser(',')

	for _, line := range lines {
		downloader.work(line)
	}

	if downloader.get_result() == "meme_url_3" {
		fmt.Println("OK")
	} else {
		fmt.Printf("ERROR: %v", downloader.get_result())
	}
}
