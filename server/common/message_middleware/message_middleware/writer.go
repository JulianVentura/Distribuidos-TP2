package message_middleware

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"sync"
	"time"

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
	batch_table        BatchTable
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

	self.batch_table = CreateBatchTable(self.encode_and_send, uint(1_000_000)) //1Mb
	self.finished.Add(1)

	go self.run()
	log.Debugf("Writer of queue %v started", self.queue_name)

	return self
}

func (self *WriteWorker) encode_and_send(topic string, messages []string) {
	parser := utils.CustomParser(rune(0x1e))
	msg := Message{
		Body:  parser.Write(messages),
		Topic: topic,
	}

	if err := self.send_message(msg); err != nil {
		log.Errorf("Error sending message on writer of %v", self.queue_name)
		return
	}
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
	var timeout <-chan time.Time
Loop:
	for {
		select {
		case <-timeout:
			self.batch_table.Flush()
		case msg, more := <-self.input:
			if self.batch_table.Is_empty() {
				timeout = time.After(time.Millisecond * 10)
			}
			if !more {
				break Loop
			}
			self.batch_table.Add_entry(msg.Topic, msg.Body)
		case <-self.quit:
			self.send_last_messages()
			break Loop
		}
	}

	log.Debugf("Writer of queue %v finishing", self.queue_name)
	self.batch_table.Flush()
	self.notify_finish()
	self.finished.Done()
}

func (self *WriteWorker) send_last_messages() {

Loop:
	for {
		select {
		case msg := <-self.input:
			self.batch_table.Add_entry(msg.Topic, msg.Body)
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
