package client

import (
	Err "distribuidos/tp2/common/errors"
	"distribuidos/tp2/common/protocol"
	"distribuidos/tp2/common/socket"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type ClientConfig struct {
	Id                    uint
	LogLevel              string
	ServerAddress         string
	LoopPeriodPost        time.Duration
	LoopPeriodComment     time.Duration
	FilePathPost          string
	FilePathComment       string
	FilePathSentimentMeme string
	FilePathSchoolMemes   string
}

type Client struct {
	server         *socket.TCPConnection
	config         ClientConfig
	postsSent      uint
	commentsSent   uint
	toSend         chan protocol.Encodable
	serverResponse chan protocol.Encodable
	finished       *sync.WaitGroup
	quitControl    []chan bool
}

func Start(config ClientConfig) (*Client, error) {
	server, err := socket.NewClient(config.ServerAddress)
	if err != nil {
		msg := fmt.Sprintf("Connection with server on address %v failed", config.ServerAddress)
		return nil, Err.Ctx(msg, err)
	}

	log.Infof("Connection with server established")
	self := &Client{
		server:         server,
		config:         config,
		postsSent:      0,
		commentsSent:   0,
		toSend:         make(chan protocol.Encodable, 100),
		serverResponse: make(chan protocol.Encodable, 2),
		quitControl:    make([]chan bool, 0, 10),
		finished:       &sync.WaitGroup{},
	}

	err = self.startReaders(&config)
	if err != nil {
		server.Close()
		return nil, err
	}

	receiveQuit := self.newQuit()
	self.finished.Add(1)
	go self.receiveFromServer(receiveQuit)

	runQuit := self.newQuit()
	self.finished.Add(1)
	go self.run(runQuit)

	return self, nil
}

func (self *Client) startReaders(config *ClientConfig) error {
	postChann := make(chan string, 100)
	commentChann := make(chan string, 100)
	postQuit := self.newQuit()
	err := readFromFile(
		config.FilePathPost,
		config.LoopPeriodPost,
		postChann,
		postQuit,
		self.finished,
	)
	if err != nil {
		self.finish()
		return err
	}
	commentQuit := self.newQuit()
	err = readFromFile(
		config.FilePathComment,
		config.LoopPeriodComment,
		commentChann,
		commentQuit,
		self.finished,
	)
	if err != nil {
		self.finish()
		return err
	}

	streamFinished := &sync.WaitGroup{}
	streamFinished.Add(2)
	go func() {
		streamFinished.Wait()
		close(self.toSend)
	}()

	go func() {
		for m := range postChann {
			log.Debugf("Sending post %s", m)
			self.postsSent += 1
			if self.postsSent%10000 == 0 {
				log.Infof("Se enviaron %v posts", self.postsSent)
			}
			self.toSend <- &protocol.Post{
				Post: m,
			}
		}
		self.toSend <- &protocol.PostFinished{}
		streamFinished.Done()
	}()

	go func() {
		for m := range commentChann {
			log.Debugf("Sending comment %s", m)
			self.commentsSent += 1
			if self.commentsSent%10000 == 0 {
				log.Infof("Se enviaron %v comments", self.commentsSent)
			}
			self.toSend <- &protocol.Comment{
				Comment: m,
			}
		}
		self.toSend <- &protocol.CommentFinished{}
		streamFinished.Done()
	}()

	return nil
}

func (self *Client) newQuit() chan bool {
	quit := make(chan bool, 1000)
	self.quitControl = append(self.quitControl, quit)
	return quit
}

func (self *Client) finish() {
	for _, quit := range self.quitControl {
		quit <- true
	}
}

func (self *Client) Finish() {
	self.finish()
	self.finished.Wait()
}

func (self *Client) run(quit chan bool) {
	streamFinished := false
	defer func() {
		if !streamFinished {
			protocol.Send(self.server, &protocol.Error{})
		}
		self.server.Close()
		self.finished.Done()
		self.finish()
		//Clean queue to prevent deadlock
		for range self.toSend {
		}
		self.finished.Wait()
	}()
Loop:
	for {
		select {
		case m, more := <-self.toSend:
			if !more {
				streamFinished = true
				self.waitForServerResponse(quit)
				break Loop
			}
			err := protocol.Send(self.server, m)
			if err != nil {
				log.Error(Err.Ctx("Error sending to server. ", err))
				break Loop
			}
		case m := <-self.serverResponse:
			should_finish := self.parseServerResponse(m)
			if should_finish {
				break Loop
			}
		case <-quit:
			break Loop
		}
	}
}

func (self *Client) waitForServerResponse(quit chan bool) {
	select {
	case m := <-self.serverResponse:
		self.parseServerResponse(m)
	case <-quit:
	}
}

func readFromFile(
	path string,
	sleepTime time.Duration,
	output chan string,
	quit chan bool,
	finished *sync.WaitGroup) error {

	f, err := os.Open(path)
	if err != nil {
		return Err.Ctx("Error opening file", err)
	}
	finished.Add(1)
	go func() {
		defer func() {
			f.Close()
			close(output)
			finished.Done()
		}()
		sep := rune(0x1f)
		reader := csv.NewReader(f)
		reader.Comma = sep
		reader.LazyQuotes = true
		reader.Read() //Extract header
	Loop:
		for {
			select {
			case <-quit:
				break Loop
			default:
				records, err := reader.Read()
				if err == io.EOF {
					break Loop
				}
				if err != nil {
					log.Errorf("Error trying to read from input file %v: %v", path, err)
					break
				}
				log.Debugf("Reading %s", strings.Join(records, ","))
				output <- strings.Join(records, string(sep))
				time.Sleep(sleepTime)
			}
		}
	}()

	return nil
}

func (self *Client) receiveFromServer(quit chan bool) {
	defer self.finished.Done()
	select {
	case <-quit:
		return
	default:
		response, err := protocol.Receive(self.server)
		if err != nil {
			log.Error(Err.Ctx("Error receiving response from server", err))
			self.finish()
			return
		}
		self.serverResponse <- response
		return
	}
}

func (self *Client) parseServerResponse(response protocol.Encodable) bool {

	switch t := response.(type) {
	case *protocol.Response:
		self.processServerResponse(t)
	case *protocol.Error:
		fmt.Printf("Error received from server: %v\n", t.Message)
	}

	return true
}

func (self *Client) processServerResponse(response *protocol.Response) {
	log.Infof("Calculation Results: ")
	log.Infof(" - Post Score AVG: %v", response.PostScoreAvg)
	log.Infof(" - Best AVG Sentiment Meme downloaded to %v", self.config.FilePathSentimentMeme)

	if len(response.SchoolMemes) > 0 {
		log.Infof(" - School Memes written to %v", self.config.FilePathSchoolMemes)
		saveSchoolMemes(response.SchoolMemes, self.config.FilePathSchoolMemes)
	} else {
		log.Infof(" - School Memes weren't found")
	}

	if len(response.BestSentimentMeme) > 0 {
		log.Infof(" - Best Sentiment Meme written to %v", self.config.FilePathSentimentMeme)
		saveSentimentMeme(response.BestSentimentMeme, self.config.FilePathSentimentMeme)
	} else {
		log.Infof(" - Best Sentiment Meme wasn't found")
	}
}

func saveSchoolMemes(memes []string, path string) {
	err := save([]byte(strings.Join(memes, "\n")), path)
	if err != nil {
		log.Errorf("Couldn't write school memes: %v", err)
	}
}

func saveSentimentMeme(meme []byte, path string) {
	err := save(meme, path)
	if err != nil {
		log.Errorf("Couldn't save meme: %v", err)
	}
}

func save(file []byte, path string) error {
	// Create the file
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer out.Close()

	return writeAll(out, file)
}

func writeAll(out io.Writer, toWrite []byte) error {
	size := len(toWrite)
	written := 0
	for written < size {
		n, err := out.Write(toWrite[written:])
		if err != nil {
			return err
		}
		written += n
	}

	return nil
}
