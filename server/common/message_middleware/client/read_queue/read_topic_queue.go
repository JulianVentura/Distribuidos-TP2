package read_queue

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	"fmt"

	"github.com/streadway/amqp"
)

type ReadTopicQueue struct {
	channel  *amqp.Channel
	delivery <-chan amqp.Delivery
	config   *mom.QueueConfig
}

func NewReadTopicQueue(
	config *mom.QueueConfig,
	channel *amqp.Channel,
) (*ReadTopicQueue, error) {

	self := &ReadTopicQueue{
		config:  config,
		channel: channel,
	}

	err := self.createQueue()
	if err != nil {
		return self, err
	}

	return self, nil
}

func (self *ReadTopicQueue) Read() <-chan amqp.Delivery {
	return self.delivery
}

func (self *ReadTopicQueue) Close() {
	self.channel.Close()
}

func (self *ReadTopicQueue) createQueue() error {
	if self.config.Topic == "" || self.config.Source == "" {
		return fmt.Errorf("Error trying to create a read topic queue: topic or source not provided")
	}

	amqpChann, err := initializeQueueAndExchange(
		self.channel,
		self.config.Source,
		"topic",
		self.config.Name,
		self.config.Topic)

	if err != nil {
		return fmt.Errorf("Error trying to create a read topic queue %v: %v", self.config.Source, err)
	}

	self.delivery = amqpChann

	return nil
}
