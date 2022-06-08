package read_queue

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	admin "distribuidos/tp2/server/common/message_middleware/client/admin_proxy"
	"distribuidos/tp2/server/common/message_middleware/client/util"
	"fmt"

	"github.com/streadway/amqp"
)

type ReadWorkerQueue struct {
	channel    *amqp.Channel
	delivery   <-chan amqp.Delivery
	config     *mom.QueueConfig
	adminProxy *admin.AdminProxy
}

func NewReadWorkerQueue(
	config *mom.QueueConfig,
	channel *amqp.Channel,
	admin *admin.AdminProxy) (*ReadWorkerQueue, error) {

	self := &ReadWorkerQueue{
		config:     config,
		channel:    channel,
		adminProxy: admin,
	}

	err := self.createQueue()
	if err != nil {
		return self, err
	}

	return self, nil
}

func (self *ReadWorkerQueue) Read() <-chan amqp.Delivery {
	return self.delivery
}

func (self *ReadWorkerQueue) Close() {
	self.channel.Close()
}

func (self *ReadWorkerQueue) createQueue() error {
	var queueOrigin string
	if self.config.Topic != "" {
		err := self.createBindQueue()
		if err != nil {
			return err
		}
		queueOrigin = self.config.Source
	} else {
		err := self.createSimpleQueue()
		if err != nil {
			return err
		}
		queueOrigin = self.config.Name
	}

	err := self.adminProxy.NewReadQueue(queueOrigin)
	if err != nil {
		return fmt.Errorf("Error notifyng new read queue to admin: %v", err)
	}

	return nil
}

func (self *ReadWorkerQueue) createBindQueue() error {
	if self.config.Source == "" {
		return fmt.Errorf("Error trying to create a bind read worker queue: source not provided")
	}
	amqpChann, err := initializeQueueAndExchange(
		self.channel,
		self.config.Source,
		"topic",
		self.config.Name,
		self.config.Topic)
	if err != nil {
		return fmt.Errorf("Error trying to create a read worker queue %v: %v", self.config.Name, err)
	}

	self.delivery = amqpChann

	return nil
}

func (self *ReadWorkerQueue) createSimpleQueue() error {
	//Declare a queue with config.name name
	queue, err := util.QueueDeclare(self.channel, self.config.Name, false)
	if err != nil {
		return fmt.Errorf("Couldn't declare the queue: %v", err)
	}
	//Get a channel from the queue
	self.delivery, err = util.ConsumeQueue(self.channel, queue.Name)
	if err != nil {
		return fmt.Errorf("Couldn't bind queue to exchange: %v", err)
	}

	return nil
}
