package write_queue

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	admin "distribuidos/tp2/server/common/message_middleware/client/admin_proxy"
	util "distribuidos/tp2/server/common/message_middleware/client/util"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type WriteTopicQueue struct {
	channel    *amqp.Channel
	config     *mom.QueueConfig
	adminProxy *admin.AdminProxy
}

func NewWriteTopicQueue(
	config *mom.QueueConfig,
	channel *amqp.Channel,
	admin *admin.AdminProxy) (*WriteTopicQueue, error) {

	self := &WriteTopicQueue{
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

func (self *WriteTopicQueue) Write(message []byte, topic string) error {

	err := util.Publish(message, self.config.Name, topic, self.channel)

	if err != nil {
		return fmt.Errorf("Error trying to publish a message to mom: %v", err)
	}
	return nil
}

func (self *WriteTopicQueue) Close() {
	err := self.adminProxy.CloseQueue(self.config.Name)
	if err != nil {
		log.Errorf("Couldn't notify admin of closed queue: %v", err)
		return
	}
	self.channel.Close()
}

func (self *WriteTopicQueue) createQueue() error {
	if self.config.Name == "" {
		return fmt.Errorf("Error trying to create a write topic queue: name not provided")
	}

	//Declare a new queue with the name 'name'
	err := util.ExchangeDeclare(self.channel, self.config.Name, "topic")
	if err != nil {
		return fmt.Errorf("Error trying to create a write topic queue %v: %v", self.config.Name, err)
	}

	err = self.adminProxy.NewWriteQueue(self.config)
	if err != nil {
		return fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	return nil
}
