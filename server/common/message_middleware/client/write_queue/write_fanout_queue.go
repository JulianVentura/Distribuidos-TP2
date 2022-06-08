package write_queue

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	admin "distribuidos/tp2/server/common/message_middleware/client/admin_proxy"
	util "distribuidos/tp2/server/common/message_middleware/client/util"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type WriteFanoutQueue struct {
	channel    *amqp.Channel
	config     *mom.QueueConfig
	adminProxy *admin.AdminProxy
}

func NewWriteFanoutQueue(
	config *mom.QueueConfig,
	channel *amqp.Channel,
	admin *admin.AdminProxy) (*WriteFanoutQueue, error) {

	self := &WriteFanoutQueue{
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

func (self *WriteFanoutQueue) Write(message []byte, topic string) error {

	err := util.Publish(message, self.config.Name, "", self.channel)

	if err != nil {
		return fmt.Errorf("Error trying to publish a message to mom: %v", err)
	}
	return nil
}

func (self *WriteFanoutQueue) Close() {
	err := self.adminProxy.CloseQueue(self.config.Name)
	if err != nil {
		log.Errorf("Couldn't notify admin of closed queue: %v", err)
		return
	}
	self.channel.Close()
}

func (self *WriteFanoutQueue) createQueue() error {
	if self.config.Name == "" {
		return fmt.Errorf("Error trying to create a write fanout queue: name not provided")
	}

	//Declare a new queue with the name 'name'
	err := util.ExchangeDeclare(self.channel, self.config.Name, "fanout")
	if err != nil {
		return fmt.Errorf("Error trying to create a write fanout queue %v: %v", self.config.Name, err)
	}

	err = self.adminProxy.NewWriteQueue(self.config)
	if err != nil {
		return fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	return nil
}
