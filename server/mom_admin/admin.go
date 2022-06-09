package main

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	momCli "distribuidos/tp2/server/common/message_middleware/client"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type MiddlewareAdmin struct {
	mom     *momCli.MessageMiddleware
	control chan mom.Message
	table   ControlTable
	quit    chan bool
}

func StartAdmin(
	momAddress string,
	quit chan bool,
) (*MiddlewareAdmin, error) {

	momConfig := momCli.MessageMiddlewareConfig{
		Address:     momAddress,
		NotifyAdmin: false,
	}

	msgMiddleware, err := momCli.Start(momConfig)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to mom: %v", err)
	}

	control, err := msgMiddleware.ReadQueue(mom.QueueConfig{
		Name:  "admin_control",
		Class: "worker",
	})
	if err != nil {
		msgMiddleware.Finish()
		return nil, fmt.Errorf("Couldn't declare control queue: %v", err)
	}

	log.Infof("Message Middleware Admin started")

	return &MiddlewareAdmin{
		mom:     msgMiddleware,
		control: control,
		table:   NewControlTable(),
		quit:    quit,
	}, nil
}

func (self *MiddlewareAdmin) Run() {

Loop:
	for {
		select {
		case <-self.quit:
			self.waitForQueuesToFinish()
			break Loop
		case m := <-self.control:
			params := strings.SplitN(m.Body, ",", 2)
			switch params[0] {
			case "new_write":
				self.handleNewWriteQueue(params[1])
			case "new_read":
				self.handleNewReadQueue(params[1])
			case "finish":
				self.handleNewQueueFinish(params[1])
			default:
				log.Errorf("Received invalid message. Topic: %v, Body: %v", m.Topic, m.Body)
			}
		}
	}

	self.mom.Finish()
	log.Infof("Middleware Admin has finished")
}

func (self *MiddlewareAdmin) waitForQueuesToFinish() {

Loop:
	for {
		if !self.table.AnyPendingFinish() {
			break Loop
		}

		select {
		case m := <-self.control:
			params := strings.SplitN(m.Body, ",", 2)
			switch params[0] {
			case "new_write":
				self.handleNewWriteQueue(params[1])
			case "new_read":
				self.handleNewReadQueue(params[1])
			case "finish":
				self.handleNewQueueFinish(params[1])
			default:
				log.Errorf("Received invalid message. Topic: %v, Body: %v", m.Topic, m.Body)
			}
		}
	}
}

func (self *MiddlewareAdmin) handleNewWriteQueue(config string) {
	log.Debugf("New Queue Writer: %v", config)
	//Parse the message
	params := strings.Split(config, ",")
	if len(params) != 5 {
		log.Errorf("Received new queue with invalid params: %v", config)
		return
	}
	qConfig := mom.QueueConfig{
		Name:      params[0],
		Class:     params[1],
		Topic:     params[2],
		Source:    params[3],
		Direction: params[4],
	}
	//Declare the new queue
	queue, err := self.mom.WriteQueue(qConfig)
	if err != nil {
		log.Errorf("Couldn't initialize queue with config %v: %v", config, err)
		return
	}
	//Introduce the new queue into the table, with the finish callback
	self.table.AddNewWriter(qConfig.Name, func(readerCount uint) {
		log.Debugf("Callback of %v has been called for %v readers", qConfig.Name, readerCount)
		//At least one message will be sent
		queue <- mom.Message{
			Topic: "finish",
			Body:  "finish",
		}
		for i := uint(1); i < readerCount; i++ {
			queue <- mom.Message{
				Topic: "finish",
				Body:  "finish",
			}
		}
	})
}

func (self *MiddlewareAdmin) handleNewReadQueue(queueName string) {
	log.Debugf("New Queue Reader: %v", queueName)
	self.table.AddNewReader(queueName)
}

func (self *MiddlewareAdmin) handleNewQueueFinish(queueName string) {
	log.Debugf("Queue finish: %v", queueName)
	self.table.DecreaseWriter(queueName)
}
