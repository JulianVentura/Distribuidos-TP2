package message_middleware

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

type WriteWorker struct {
	queue_name         string
	exchange           string
	routing_key_by_msg bool
	routing_key        string
	input              chan Message
	output             *amqp.Channel
	notify_on_finish   bool
	quit               chan bool
	finished           sync.WaitGroup
}

func writer_worker(
	queue_name string,
	exchange string,
	routing_key_by_msg bool,
	routing_key string,
	input chan Message,
	output *amqp.Channel,
	notify_on_finish bool) *WriteWorker {

	self := &WriteWorker{
		queue_name:         queue_name,
		exchange:           exchange,
		routing_key_by_msg: routing_key_by_msg,
		routing_key:        routing_key,
		input:              input,
		output:             output,
		quit:               make(chan bool, 2),
		notify_on_finish:   notify_on_finish,
	}

	self.finished.Add(1)
	go self.run()
	log.Debugf("Writer of queue %v started", self.queue_name)

	return self
}

func (self *WriteWorker) finish() {
	// if self.input != nil {
	// 	close(self.input)
	// 	self.input = nil
	// }
	self.quit <- true
	self.finished.Wait()
}

func (self *WriteWorker) run() {

Loop:
	for {
		select {
		case msg, more := <-self.input:
			if !more {
				break Loop
			}
			if err := self.send_message(msg); err != nil {
				log.Errorf("Error sending message on writer of %v", self.queue_name)
				return
			}
		case <-self.quit:
			self.send_last_messages()
			break Loop
		}
	}

	log.Debugf("Writer of queue %v finishing", self.queue_name)
	self.notify_finish()
	self.finished.Done()
}

func (self *WriteWorker) send_last_messages() {

Loop:
	for {
		select {
		case msg := <-self.input:
			if err := self.send_message(msg); err != nil {
				log.Errorf("Error sending message on writer of %v", self.queue_name)
				return
			}
		default:
			break Loop
		}
	}
}

func (self *WriteWorker) notify_finish() {

	if !self.notify_on_finish {
		return
	}

	target_queue := "admin_control"
	message := fmt.Sprintf("finish,%v", self.queue_name)

	err := self.output.Publish(
		"",           // exchange
		target_queue, // routing key
		false,        // mandatory
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})

	if err != nil {
		log.Errorf("Error trying to notify mom admin: %v", err)
	}
}

func (self *WriteWorker) send_message(msg Message) error {
	var rout_key string
	if self.routing_key_by_msg {
		rout_key = msg.Topic
	} else {
		rout_key = self.routing_key
	}
	err := self.output.Publish(
		self.exchange, // exchange
		rout_key,      // routing key
		false,         // mandatory
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg.Body),
		})

	if err != nil {
		log.Errorf("Error trying to publish a message to mom: %v", err)
		return err
	}

	return nil
}
