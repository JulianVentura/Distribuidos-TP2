package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type BestSentimentAvgDownloader struct {
	Best_score float64
	Best       string
}

func NewDownloader() BestSentimentAvgDownloader {
	return BestSentimentAvgDownloader{
		Best_score: math.Inf(-1),
		Best:       "",
	}
}

func (self *BestSentimentAvgDownloader) work(input string) {
	log.Debugf("Received: %v", input)

	split := strings.Split(input, ",")
	score, _ := strconv.ParseFloat(split[3], 64)

	if score > self.Best_score {
		self.Best_score = score
		self.Best = split[1]
	}
}

func (self *BestSentimentAvgDownloader) get_result() string {
	return fmt.Sprintf("%v", self.Best)
}

func test_function() {
	lines := []string{
		"a,meme_url_1,2,0.23",
		"b,meme_url_2,2,0.0",
		"c,meme_url_3,2,0.87",
		"d,meme_url_4,2,-0.89",
	}

	downloader := NewDownloader()

	for _, line := range lines {
		downloader.work(line)
	}

	if downloader.get_result() == "meme_url_3" {
		fmt.Println("OK")
	} else {
		fmt.Printf("ERROR: %v", downloader.get_result())
	}
}
