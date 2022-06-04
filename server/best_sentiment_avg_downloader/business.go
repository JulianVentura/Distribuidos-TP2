package main

import (
	"distribuidos/tp2/server/common/utils"
	"io"
	"math"
	"net/http"
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
	if len(split) != 2 {
		log.Errorf("Received bad formated input")
		return
	}
	score, err := strconv.ParseFloat(split[1], 64)
	if err != nil {
		log.Errorf("Couldn't parse score: %v", err)
		return
	}

	if score > self.Best_score {
		self.Best_score = score
		self.Best = split[0]
	}
}

func (self *BestSentimentAvgDownloader) get_result() string {
	resp, err := http.Get(self.Best)
	if err != nil {
		log.Errorf("Error getting meme of url %v: %v", self.Best, err)
		return ""
	}
	defer resp.Body.Close()

	file, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error downloading meme of url %v: %v", self.Best, err)
		return ""
	}
	return string(file) //We can "see" the byte slice as string and vice versa without loosing any information
}
