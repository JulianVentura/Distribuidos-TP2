package client

import (
	"fmt"
	"time"

	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/message_middleware/client/admin_proxy"
	rq "distribuidos/tp2/server/common/message_middleware/client/read_queue"
	workers "distribuidos/tp2/server/common/message_middleware/client/workers"
	wq "distribuidos/tp2/server/common/message_middleware/client/write_queue"

	"github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

type MessageMiddlewareConfig struct {
	Address                string
	NotifyAdmin            bool
	ChannelBufferSize      uint
	MessageBatchSizeTarget uint
	MessageBatchTimeout    time.Duration
}

type MessageMiddleware struct {
	connection *amqp.Connection
	writers    []*workers.WriteWorker
	readers    []*workers.ReadWorker
	config     MessageMiddlewareConfig
}

const WRITER_WORKERS_INIT_SIZE = 1
const READER_WORKERS_INIT_SIZE = 1

func Start(config MessageMiddlewareConfig) (*MessageMiddleware, error) {

	self := &MessageMiddleware{
		writers: make([]*workers.WriteWorker, 0, WRITER_WORKERS_INIT_SIZE),
		readers: make([]*workers.ReadWorker, 0, READER_WORKERS_INIT_SIZE),
		config:  config,
	}

	conn, err := amqp.Dial(config.Address)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
	}

	self.connection = conn

	return self, nil
}

func (self *MessageMiddleware) newChannel() (*amqp.Channel, error) {
	ch, err := self.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Failed to open a channel: %v", err)
	}

	err = ch.Qos(1, 0, true)

	if err != nil {
		return nil, err
	}

	return ch, nil
}

func (self *MessageMiddleware) Finish() error {
	logrus.Debugf("Closing readers")
	for _, reader := range self.readers {
		reader.Finish()
	}

	logrus.Debugf("Closing writers")
	for _, writer := range self.writers {
		writer.Finish()
	}

	logrus.Debugf("Closing connection")
	self.connection.Close()

	return nil
}

func (self *MessageMiddleware) NewQueue(config mom.QueueConfig) (chan mom.Message, error) {
	switch config.Direction {
	case "read":
		return self.ReadQueue(config)
	case "write":
		return self.WriteQueue(config)
	default:
		return nil, fmt.Errorf("Error trying to create a queue: Direction %v is not recognized", config.Direction)
	}
}

func (self *MessageMiddleware) ReadQueue(config mom.QueueConfig) (chan mom.Message, error) {
	var err error = nil
	var queue rq.ReadQueue = nil

	channel := make(chan mom.Message, self.config.ChannelBufferSize)
	ch, err := self.newChannel()
	if err != nil {
		return nil, err
	}
	switch config.Class {
	case "worker":
		admin, err := admin_proxy.NewAdminProxy(ch, self.config.NotifyAdmin)
		if err != nil {
			return nil, err
		}
		queue, err = rq.NewReadWorkerQueue(&config, ch, &admin)
	case "topic":
		queue, err = rq.NewReadTopicQueue(&config, ch)
	case "fanout":
		queue, err = rq.NewReadFanoutQueue(&config, ch)
	default:
		return nil, fmt.Errorf("Error trying to create a read queue: Class %v is not recognized", config.Class)
	}

	worker := workers.StartReadWorker(queue, channel)

	self.readers = append(self.readers, worker)

	return channel, nil
}

func (self *MessageMiddleware) WriteQueue(config mom.QueueConfig) (chan mom.Message, error) {
	var err error = nil
	var queue wq.WriteQueue = nil

	channel := make(chan mom.Message, self.config.ChannelBufferSize)
	ch, err := self.newChannel()
	if err != nil {
		return nil, err
	}
	admin, err := admin_proxy.NewAdminProxy(ch, self.config.NotifyAdmin)
	if err != nil {
		return nil, err
	}
	switch config.Class {
	case "worker":
		queue, err = wq.NewWriteWorkerQueue(&config, ch, &admin)
	case "topic":
		queue, err = wq.NewWriteTopicQueue(&config, ch, &admin)
	case "fanout":
		queue, err = wq.NewWriteFanoutQueue(&config, ch, &admin)
	default:
		return nil, fmt.Errorf("Error trying to create a write queue: Class %v is not recognized", config.Class)
	}

	wConfig := workers.WriteWorkerConfig{
		MessageBatchSizeTarget: self.config.MessageBatchSizeTarget,
		MessageBatchTimeout:    self.config.MessageBatchTimeout,
	}

	worker := workers.StartWriteWorker(wConfig, channel, queue)

	self.writers = append(self.writers, worker)

	return channel, nil
}
