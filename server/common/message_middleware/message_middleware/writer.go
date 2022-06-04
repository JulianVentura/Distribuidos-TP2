package message_middleware

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

type WriteWorker struct {
	queueName       string
	exchange        string
	routingKeyByMsg bool
	routingKey      string
	input           chan Message
	output          *amqp.Channel
	notifyOnFinish  bool
	batchTable      BatchTable
	quit            chan bool
	finished        sync.WaitGroup
}

func writerWorker(
	queueName string,
	exchange string,
	routingKeyBybsg bool,
	routingKey string,
	input chan Message,
	output *amqp.Channel,
	notifyOnFinish bool) *WriteWorker {

	self := &WriteWorker{
		queueName:       queueName,
		exchange:        exchange,
		routingKeyByMsg: routingKeyBybsg,
		routingKey:      routingKey,
		input:           input,
		output:          output,
		quit:            make(chan bool, 2),
		notifyOnFinish:  notifyOnFinish,
	}

	//TODO: Config
	self.batchTable = createBatchTable(self.encodeAndSend, uint(1_000_000)) //1Mb
	self.finished.Add(1)

	go self.run()
	log.Debugf("Writer of queue %v started", self.queueName)

	return self
}

func (self *WriteWorker) encodeAndSend(topic string, messages []string) {
	if err := self.sendMessages(messages, topic); err != nil {
		log.Errorf("Error sending message on writer of %v", self.queueName)
		return
	}
}

func (self *WriteWorker) finish() {
	self.quit <- true
	self.finished.Wait()
}

func (self *WriteWorker) run() {
	var timeout <-chan time.Time
Loop:
	for {
		select {
		case <-timeout:
			self.batchTable.flush()
		case msg, more := <-self.input:
			if self.batchTable.isEmpty() {
				timeout = time.After(time.Millisecond * 10) //TODO: Config
			}
			if !more {
				break Loop
			}
			self.batchTable.addEntry(msg.Topic, msg.Body)
		case <-self.quit:
			self.sendLastMessages()
			break Loop
		}
	}

	log.Debugf("Writer of queue %v finishing", self.queueName)
	self.batchTable.flush()
	self.notifyFinish()
	self.finished.Done()
}

func (self *WriteWorker) sendLastMessages() {

Loop:
	for {
		select {
		case msg := <-self.input:
			self.batchTable.addEntry(msg.Topic, msg.Body)
		default:
			break Loop
		}
	}
}

func (self *WriteWorker) notifyFinish() {

	if !self.notifyOnFinish {
		return
	}

	targetQueue := "admin_control"
	msg := fmt.Sprintf("finish,%v", self.queueName)
	err := publish([]string{msg}, "", targetQueue, self.output)
	if err != nil {
		log.Errorf("Error trying to notify mom admin: %v", err)
	}
}

func (self *WriteWorker) sendMessages(messages []string, topic string) error {
	var routKey string
	if self.routingKeyByMsg {
		routKey = topic
	} else {
		routKey = self.routingKey
	}
	return publish(messages, self.exchange, routKey, self.output)
}
