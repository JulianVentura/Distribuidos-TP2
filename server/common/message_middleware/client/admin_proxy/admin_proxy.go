package admin_proxy

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/message_middleware/client/protocol"
	util "distribuidos/tp2/server/common/message_middleware/client/util"
	"fmt"

	amqp "github.com/streadway/amqp"
)

type AdminProxy struct {
	notify  bool
	channel *amqp.Channel
}

func NewAdminProxy(
	chann *amqp.Channel,
	notify bool,
) (AdminProxy, error) {

	_, err := util.QueueDeclare(chann, "admin_control", false)
	if err != nil {
		return AdminProxy{}, fmt.Errorf("Couldn't declare admin's queue: %v", err)
	}
	return AdminProxy{
		notify:  notify,
		channel: chann,
	}, nil
}

func (self *AdminProxy) NewWriteQueue(config *mom.QueueConfig) error {
	message := fmt.Sprintf("new_write,%v,%v,%v,%v,%v",
		config.Name,
		config.Class,
		config.Topic,
		config.Source,
		config.Direction)

	return self.notifyToAdmin(message)
}

func (self *AdminProxy) NewReadQueue(queueId string) error {
	message := fmt.Sprintf("new_read,%v", queueId)
	return self.notifyToAdmin(message)
}

func (self *AdminProxy) CloseQueue(queueId string) error {
	msg := fmt.Sprintf("finish,%v", queueId)
	return self.notifyToAdmin(msg)
}

func (self *AdminProxy) notifyToAdmin(message string) error {
	if !self.notify {
		return nil
	}
	targetQueue := "admin_control"
	msg := []byte(message)
	encoded := protocol.EncodeBytesSlice([][]byte{msg})

	err := util.Publish(encoded, "", targetQueue, self.channel)

	return err
}
