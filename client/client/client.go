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
	posts_queue     chan string
	comments_queue  chan string
	server_response chan protocol.Encodable
	posts_finish    chan bool
	comments_finish chan bool
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
		posts_queue:     make(chan string, 100),
		comments_queue:  make(chan string, 100),
		server_response: make(chan protocol.Encodable, 2),
		posts_finish:    make(chan bool, 2),
		comments_finish: make(chan bool, 2),
		quit:            make(chan bool, 2),
		has_finished:    make(chan bool, 2),
	}

	err = read_from_file(
		config.File_path_post,
		config.Loop_period_post,
		client.posts_queue,
		client.posts_finish,
		client.quit,
	)
	if err != nil {
		client.quit <- true
		return nil, err
	}
	// err = read_from_file(
	// 	config.File_path_comment,
	// 	config.Loop_period_comment,
	// 	client.comments_queue,
	// 	client.comments_finish,
	// 	client.quit,
	// )
	// if err != nil {
	// 	client.quit <- true //We close previous thread
	// 	return nil, err
	// }

	go client.receive_from_server()
	go client.run()

	return client, nil
}

func (self *Client) Finish() {
	self.quit <- true
	<-self.has_finished
}

func (self *Client) run() {
	defer func() {
		self.server.Close()
		self.has_finished <- true
	}()

Loop:
	for {
		select {
		case p := <-self.posts_queue:
			if err := self.send_post(p); err != nil {
				return
			}
		case c := <-self.comments_queue:
			if err := self.send_comment(c); err != nil {
				return
			}
		case <-self.comments_finish:
			if err := self.send_comments_finished(); err != nil {
				return
			}
		case <-self.posts_finish:
			if err := self.send_posts_finished(); err != nil {
				return
			}
		case r := <-self.server_response:
			should_finish := parse_server_response(r)
			if should_finish {
				return
			}
		case <-self.quit:
			break Loop
		}
	}
}

func (self *Client) send_post(post string) error {
	log.Debugf("Sending post %s", post)
	err := protocol.Send(self.server, &protocol.Post{Post: post})
	if err != nil {
		log.Error(Err.Ctx("Error sending a post to server. ", err))
		return err
	}
	self.posts_sent += 1
	if self.posts_sent%10000 == 0 {
		log.Infof("Se enviaron %v posts", self.posts_sent)
	}

	return nil
}

func (self *Client) send_comment(comment string) error {
	log.Debugf("Sending comment %s", comment)
	err := protocol.Send(self.server, &protocol.Comment{Comment: comment})
	if err != nil {
		log.Error(Err.Ctx("Error sending a comment to server. ", err))
		return err
	}
	self.comments_sent += 1
	if self.posts_sent%10000 == 0 {
		log.Infof("Se enviaron %v comments", self.comments_sent)
	}

	return nil
}

func (self *Client) send_posts_finished() error {
	log.Infof("Se envía fin de posts")
	err := protocol.Send(self.server, &protocol.PostFinished{})
	if err != nil {
		log.Error(Err.Ctx("Error sending a PostsFinished to server. ", err))
		return err
	}

	return nil
}

func (self *Client) send_comments_finished() error {
	log.Infof("Se envía fin de comments")
	err := protocol.Send(self.server, &protocol.CommentFinished{})
	if err != nil {
		log.Error(Err.Ctx("Error sending a CommentsFinished to server. ", err))
		return err
	}

	return nil
}

func read_from_file(
	path string,
	sleep_time time.Duration,
	output chan string,
	notify_finish chan bool,
	quit chan bool) error {

	f, err := os.Open(path)
	if err != nil {
		return Err.Ctx("Error opening file", err)
	}

	go func() {
		defer f.Close()
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
		notify_finish <- true
	}()

	return nil
}

func (self *Client) receive_from_server() {

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
	log.Printf("%v", response.Post_score_average)
}
