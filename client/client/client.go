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
	finished       sync.WaitGroup
	quit           chan bool
	hasFinished    chan bool
}

func Start(config ClientConfig) (*Client, error) {
	server, err := socket.NewClient(config.ServerAddress)
	// var err error = nil
	if err != nil {
		msg := fmt.Sprintf("Connection with server on address %v failed", config.ServerAddress)
		return nil, Err.Ctx(msg, err)
	}

	log.Infof("Connection with server established")
	client := &Client{
		server:         server,
		config:         config,
		postsSent:      0,
		commentsSent:   0,
		toSend:         make(chan protocol.Encodable, 100000),
		serverResponse: make(chan protocol.Encodable, 2),
		quit:           make(chan bool, 2),
		hasFinished:    make(chan bool, 2),
		finished:       sync.WaitGroup{},
	}

	postChann := make(chan string, 10000)
	commentChann := make(chan string, 10000)

	err = readFromFile(
		config.FilePathPost,
		config.LoopPeriodPost,
		postChann,
		client.quit,
	)
	if err != nil {
		client.quit <- true
		return nil, err
	}
	err = readFromFile(
		config.FilePathComment,
		config.LoopPeriodComment,
		commentChann,
		client.quit,
	)
	if err != nil {
		client.quit <- true //We close previous thread
		return nil, err
	}

	go func() {
		for m := range postChann {
			log.Debugf("Sending post %s", m)
			client.postsSent += 1
			if client.postsSent%10000 == 0 {
				log.Infof("Se enviaron %v posts", client.postsSent)
			}
			client.toSend <- &protocol.Post{
				Post: m,
			}
		}
		log.Infof("Posts finished")
		client.toSend <- &protocol.PostFinished{}
		client.finished.Done()
	}()

	go func() {
		for m := range commentChann {
			log.Debugf("Sending comment %s", m)
			client.commentsSent += 1
			if client.commentsSent%10000 == 0 {
				log.Infof("Se enviaron %v comments", client.commentsSent)
			}
			client.toSend <- &protocol.Comment{
				Comment: m,
			}
		}
		log.Infof("Comments finished")
		client.toSend <- &protocol.CommentFinished{}
		client.finished.Done()
	}()

	client.finished.Add(2)

	go func() {
		client.finished.Wait()
		close(client.toSend)
	}()

	go client.receiveFromServer()
	go client.run()

	return client, nil
}

func (self *Client) Finish() {
	self.quit <- true
	self.finished.Wait()
	<-self.hasFinished
}

func (self *Client) run() {
	defer func() {
		self.server.Close()
		self.hasFinished <- true
	}()
Loop:
	for {
		select {
		case m, more := <-self.toSend:
			if !more {
				self.waitForServerResponse()
				break Loop
			}
			err := protocol.Send(self.server, m)
			if err != nil {
				log.Error(Err.Ctx("Error sending to server. ", err))
				break Loop
			}
		case m := <-self.serverResponse:
			should_finish := self.parseServerResponse(m)
			if should_finish { //TODO: Handle this
				break Loop
			}
		case <-self.quit:
			self.quit <- true
			break Loop
		}
	}
}

func (self *Client) waitForServerResponse() {
	select {
	case m := <-self.serverResponse:
		self.parseServerResponse(m)
	case <-self.quit:
		self.quit <- true
	}
}

func readFromFile(
	path string,
	sleepTime time.Duration,
	output chan string,
	quit chan bool) error {

	f, err := os.Open(path)
	if err != nil {
		return Err.Ctx("Error opening file", err)
	}

	go func() {
		defer func() {
			f.Close()
			close(output)
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
				quit <- true //Propagate the signal
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

func (self *Client) receiveFromServer() {

	select {
	case <-self.quit:
		self.quit <- true
		return
	default:
		response, err := protocol.Receive(self.server)
		if err != nil {
			log.Error(Err.Ctx("Error receiving response from server", err))
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
