package message_middleware

import (
	Err "distribuidos/tp2/common/errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

type MessageMiddlewareConfig struct {
	Address                string
	NotifyOnFinish         bool
	ChannelBufferSize      uint
	MessageBatchSizeTarget uint
	MessageBatchTimeout    time.Duration
}

type MessageMiddleware struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	writers    []*WriteWorker
	config     MessageMiddlewareConfig
}

type QueueConfig struct {
	Name      string
	Class     string
	Topic     string
	Source    string
	Direction string
}

const WRITER_WORKERS_INIT_SIZE = 1

func Start(config MessageMiddlewareConfig) (*MessageMiddleware, error) {

	self := &MessageMiddleware{
		writers: make([]*WriteWorker, 0, WRITER_WORKERS_INIT_SIZE),
		config:  config,
	}

	err := self.startConnection(config.Address)

	if err != nil {
		return nil, err
	}

	err = self.declareAdminQueue()
	if err != nil {
		return nil, fmt.Errorf("Error trying to declare admin queue: %v", err)
	}

	return self, nil
}

func (self *MessageMiddleware) declareAdminQueue() error {
	_, err := self.queueDeclare("admin_control", false)
	if err != nil {
		return nil
	}

	return nil
}

func (self *MessageMiddleware) startConnection(host string) error {
	conn, err := amqp.Dial(host)
	if err != nil {
		return Err.Ctx("Failed to connect to RabbitMQ", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return Err.Ctx("Failed to open a channel", err)
	}

	self.connection = conn
	self.channel = ch

	return nil
}

func (self *MessageMiddleware) Finish() error {

	for _, writer := range self.writers {
		writer.finish()
	}
	self.channel.Close()
	self.connection.Close()

	return nil
}

func (self *MessageMiddleware) NewQueue(config QueueConfig) (chan Message, error) {
	switch config.Direction {
	case "read":
		return self.ReadQueue(config)
	case "write":
		return self.WriteQueue(config)
	default:
		return nil, fmt.Errorf("Error trying to create a queue: Direction %v is not recognized", config.Direction)
	}
}

func (self *MessageMiddleware) ReadQueue(config QueueConfig) (chan Message, error) {
	switch config.Class {
	case "worker":
		return self.createReadWorkerQueue(&config)
	case "topic":
		return self.createReadTopicQueue(&config)
	case "fanout":
		return self.createReadFanoutQueue(&config)
	default:
		return nil, fmt.Errorf("Error trying to create a read queue: Class %v is not recognized", config.Class)
	}
}

func (self *MessageMiddleware) createReadWorkerQueue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.ChannelBufferSize)
	var amqpChann <-chan amqp.Delivery
	var err error

	if config.Topic != "" {
		if config.Source == "" {
			return nil, fmt.Errorf("Error trying to create a read worker queue: source not provided")
		}
		amqpChann, err = self.initializeQueueAndExchange(
			config.Source,
			"topic",
			config.Name,
			config.Topic)
		if err != nil {
			return nil, fmt.Errorf("Error trying to create a read worker queue %v: %v", config.Name, err)
		}
	} else {
		if config.Name == "" {
			return nil, fmt.Errorf("Error trying to create a read worker queue: name not provided")
		}
		//Declare a queue with config.name name
		queue, err := self.queueDeclare(config.Name, false)
		if err != nil {
			return nil, Err.Ctx("Couldn't declare the queue", err)
		}
		//Get a channel from the queue
		amqpChann, err = self.consumeQueue(queue.Name)
		if err != nil {
			return nil, Err.Ctx("Couldn't bind queue to exchange", err)
		}
	}

	//We invoke a new worker, which will listen for new messages on the queue
	go readWorker(amqpChann, channel)

	return channel, nil
}

func (self *MessageMiddleware) createReadTopicQueue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.ChannelBufferSize)
	if config.Topic == "" || config.Source == "" {
		return nil, fmt.Errorf("Error trying to create a read topic queue: topic or source not provided")
	}
	amqpChann, err := self.initializeQueueAndExchange(
		config.Source,
		"topic",
		config.Name,
		config.Topic)

	if err != nil {
		return nil, fmt.Errorf("Error trying to create a read topic queue %v: %v", config.Source, err)
	}
	//We invoke a new worker, which will listen for new messages on the queue
	go readWorker(amqpChann, channel)

	return channel, nil
}

func (self *MessageMiddleware) createReadFanoutQueue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.ChannelBufferSize)
	if config.Source == "" {
		return nil, fmt.Errorf("Error trying to create a read fanout queue: source not provided")
	}
	amqpChann, err := self.initializeQueueAndExchange(
		config.Source,
		"fanout",
		config.Name,
		"")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a read fanout queue %v: %v", config.Source, err)
	}

	//We invoke a new worker, which will listen for new messages on the queue
	go readWorker(amqpChann, channel)

	return channel, nil
}

func readWorker(input <-chan amqp.Delivery, output chan Message) {
	//Will continue until the connection is closed
	for msg := range input {
		messages := decodeStringSlice(msg.Body)
		if messages[0] == "finish" {
			err := msg.Nack(false, true)
			if err != nil {
				log.Debugf("Error making Nack of message")
			}
			log.Debugf("Reader has found finish message, closing...")
			close(output)
			return
		}
		err := msg.Ack(false)
		if err != nil {
			log.Errorf("Failed to ack message")
		}
		for _, m := range messages {
			output <- Message{
				Body:  m,
				Topic: msg.RoutingKey,
			}
		}
	}
}

func (self *MessageMiddleware) initializeQueueAndExchange(
	exchangeName string,
	exchangeType string,
	queueName string,
	queueTopic string,
) (<-chan amqp.Delivery, error) {
	//Declare an exchange with the name 'source' and the class 'topic'
	err := self.exchangeDeclare(exchangeName, exchangeType)
	if err != nil {
		return nil, Err.Ctx("Couldn't declare the exchange", err)
	}
	//Declare a new queue with the name 'name', which could be empty
	queue, err := self.queueDeclare(queueName, false)
	if err != nil {
		return nil, Err.Ctx("Couldn't declare the queue", err)
	}
	//Bind the queue to the exchange
	err = self.queueBind(queue.Name, queueTopic, exchangeName)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange", err)
	}
	//Bind the queue to the admin "finish" topic
	err = self.queueBind(queue.Name, "finish", exchangeName)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange, with topic finish", err)
	}
	//Get a channel from the queue
	amqpChann, err := self.consumeQueue(queue.Name)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange", err)
	}
	return amqpChann, nil
}

func (self *MessageMiddleware) WriteQueue(config QueueConfig) (chan Message, error) {
	switch config.Class {
	case "worker":
		return self.createWriteWorkerQueue(&config)
	case "topic":
		return self.createWriteTopicQueue(&config)
	case "fanout":
		return self.createWriteFanoutQueue(&config)
	default:
		return nil, fmt.Errorf("Error trying to create a write queue: Class %v is not recognized", config.Class)
	}
}

func (self *MessageMiddleware) createWriteWorkerQueue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.ChannelBufferSize)
	if config.Name == "" {
		return nil, fmt.Errorf("Error trying to create a write worker queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	_, err := self.queueDeclare(config.Name, false)
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write worker queue %v: %v", config.Name, err)
	}

	err = self.notifyNewWriteQueueToAdmin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	wConfig := WriteWorkerConfig{
		queueName:              config.Name,
		exchange:               "",
		routingKeyByMsg:        false,
		routingKey:             config.Name,
		notifyOnFinish:         self.config.NotifyOnFinish,
		messageBatchSizeTarget: self.config.MessageBatchSizeTarget,
		messageBatchTimeout:    self.config.MessageBatchTimeout,
	}

	writer := writerWorker(
		wConfig,
		channel,
		self.channel,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) createWriteTopicQueue(config *QueueConfig) (chan Message, error) {
	log.Debugf("Creating a work queue %v", config)
	channel := make(chan Message, self.config.ChannelBufferSize)
	if config.Name == "" {
		return nil, fmt.Errorf("Error trying to create a write topic queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	err := self.exchangeDeclare(config.Name, "topic")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write topic queue %v: %v", config.Name, err)
	}

	err = self.notifyNewWriteQueueToAdmin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	wConfig := WriteWorkerConfig{
		queueName:              config.Name,
		exchange:               config.Name,
		routingKeyByMsg:        true,
		routingKey:             "",
		notifyOnFinish:         self.config.NotifyOnFinish,
		messageBatchSizeTarget: self.config.MessageBatchSizeTarget,
		messageBatchTimeout:    self.config.MessageBatchTimeout,
	}

	writer := writerWorker(
		wConfig,
		channel,
		self.channel,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) createWriteFanoutQueue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.ChannelBufferSize)
	if config.Name == "" {
		return nil, fmt.Errorf("Error trying to create a write fanout queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	err := self.exchangeDeclare(config.Name, "fanout")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write fanout queue %v: %v", config.Name, err)
	}

	err = self.notifyNewWriteQueueToAdmin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	wConfig := WriteWorkerConfig{
		queueName:              config.Name,
		exchange:               config.Name,
		routingKeyByMsg:        false,
		routingKey:             "",
		notifyOnFinish:         self.config.NotifyOnFinish,
		messageBatchSizeTarget: self.config.MessageBatchSizeTarget,
		messageBatchTimeout:    self.config.MessageBatchTimeout,
	}

	writer := writerWorker(
		wConfig,
		channel,
		self.channel,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) exchangeDeclare(name string, class string) error {
	return self.channel.ExchangeDeclare(
		name,  // name
		class, // type
		false, // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

func (self *MessageMiddleware) queueDeclare(name string, exclusive bool) (amqp.Queue, error) {
	return self.channel.QueueDeclare(
		name,      // name
		false,     // durable
		false,     // delete when unused
		exclusive, // exclusive
		false,     // no-wait
		nil,       // arguments
	)
}

func (self *MessageMiddleware) queueBind(name string, routingKey string, exchange string) error {
	return self.channel.QueueBind(
		name,       // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
}

func (self *MessageMiddleware) consumeQueue(name string) (<-chan amqp.Delivery, error) {
	return self.channel.Consume(
		name,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (self *MessageMiddleware) notifyNewWriteQueueToAdmin(config *QueueConfig) error {

	//TODO: Cambiar el nombre de la variable a algo que tenga más sentido
	//Lo que estamos haciendo acá es no enviar el mensaje si se trata del admin
	if !self.config.NotifyOnFinish {
		return nil
	}

	targetQueue := "admin_control"
	message := fmt.Sprintf("new_write,%v,%v,%v,%v,%v",
		config.Name,
		config.Class,
		config.Topic,
		config.Source,
		config.Direction)

	err := publish([]string{message}, "", targetQueue, self.channel)

	return err
}
