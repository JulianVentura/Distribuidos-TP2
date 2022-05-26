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
	Id                  uint
	Log_level           string
	Server_address      string
	Loop_period_post    time.Duration
	Loop_period_comment time.Duration
	File_path_post      string
	File_path_comment   string
}

type Client struct {
	server          *socket.TCPConnection
	config          ClientConfig
	posts_sent      uint
	comments_sent   uint
	to_send         chan protocol.Encodable
	server_response chan protocol.Encodable
	finished        sync.WaitGroup
	quit            chan bool
	has_finished    chan bool
}

func Start(config ClientConfig) (*Client, error) {
	server, err := socket.NewClient(config.Server_address)
	// var err error = nil
	if err != nil {
		msg := fmt.Sprintf("Connection with server on address %v failed", config.Server_address)
		return nil, Err.Ctx(msg, err)
	}

	log.Infof("Connection with server established")
	client := &Client{
		server:          server,
		config:          config,
		posts_sent:      0,
		comments_sent:   0,
		to_send:         make(chan protocol.Encodable, 1000),
		server_response: make(chan protocol.Encodable, 2),
		quit:            make(chan bool, 2),
		has_finished:    make(chan bool, 2),
		finished:        sync.WaitGroup{},
	}

	post_chan := make(chan string, 100)
	comment_chan := make(chan string, 100)

	err = read_from_file(
		config.File_path_post,
		config.Loop_period_post,
		post_chan,
		client.quit,
	)
	if err != nil {
		client.quit <- true
		return nil, err
	}
	err = read_from_file(
		config.File_path_comment,
		config.Loop_period_comment,
		comment_chan,
		client.quit,
	)
	if err != nil {
		client.quit <- true //We close previous thread
		return nil, err
	}

	go func() {
		for m := range post_chan {
			log.Debugf("Sending post %s", m)
			client.posts_sent += 1
			if client.posts_sent%10000 == 0 {
				log.Infof("Se enviaron %v posts", client.posts_sent)
			}
			client.to_send <- &protocol.Post{
				Post: m,
			}
		}
		log.Infof("Posts finished")
		client.to_send <- &protocol.PostFinished{}
		client.finished.Done()
	}()

	go func() {
		for m := range comment_chan {
			log.Debugf("Sending comment %s", m)
			client.comments_sent += 1
			if client.comments_sent%10000 == 0 {
				log.Infof("Se enviaron %v comments", client.comments_sent)
			}
			client.to_send <- &protocol.Comment{
				Comment: m,
			}
		}
		log.Infof("Comments finished")
		client.to_send <- &protocol.CommentFinished{}
		client.finished.Done()
	}()

	client.finished.Add(2)

	go func() {
		client.finished.Wait()
		close(client.to_send)
	}()

	go client.receive_from_server()
	go client.run()

	return client, nil
}

func (self *Client) Finish() {
	self.quit <- true
	self.finished.Wait()
	<-self.has_finished
}

func (self *Client) run() {
	defer func() {
		self.server.Close()
		self.has_finished <- true
	}()
	//Problema: Se entra en loop cuando alguno de los dos se cierra
Loop:
	for {
		select {
		case m, more := <-self.to_send:
			if !more {
				self.wait_for_server_response()
				break Loop
			}
			err := protocol.Send(self.server, m)
			if err != nil {
				log.Error(Err.Ctx("Error sending to server. ", err))
				break Loop
			}
		case m := <-self.server_response:
			should_finish := parse_server_response(m)
			if should_finish {
				break Loop
			}
		case <-self.quit:
			self.quit <- true
			break Loop
		}
	}
}

func (self *Client) wait_for_server_response() {
	select {
	case m := <-self.server_response:
		parse_server_response(m)
	case <-self.quit:
		self.quit <- true
	}
}

func read_from_file(
	path string,
	sleep_time time.Duration,
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
		reader := csv.NewReader(f)
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
					log.Errorf("Error trying to read from input file %v", path)
					break
				}
				log.Debugf("Reading %s", strings.Join(records, ","))
				output <- strings.Join(records, ",")
				time.Sleep(sleep_time)
			}
		}
	}()

	return nil
}

func (self *Client) receive_from_server() {

	select {
	case <-self.quit:
		self.quit <- true
		return
	default:
		response, err := protocol.Receive(self.server) //TODO: Aca habria que usar timeout
		if err != nil {
			log.Error(Err.Ctx("Error receiving response from server", err))
			return
		}
		self.server_response <- response
		return
	}
}

func parse_server_response(response protocol.Encodable) bool {

	switch t := response.(type) {
	case *protocol.Response:
		print_server_response(t)
	case *protocol.Error:
		fmt.Printf("Error received from server: %v\n", t.Message)
	}

	return true
}

func print_server_response(response *protocol.Response) {
	log.Infof("Calculation Results: ")
	log.Infof(" - Post Score AVG: %v", response.Post_score_average)
	log.Infof(" - Best AVG Sentiment Meme: %v", response.Best_sentiment_meme)
}
