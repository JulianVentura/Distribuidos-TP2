package message_middleware

import (
	Err "distribuidos/tp2/common/errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

type MessageMiddlewareConfig struct {
	Address             string
	Notify_on_finish    bool
	Channel_buffer_size uint
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

func Start(config MessageMiddlewareConfig) (*MessageMiddleware, error) {

	config.Channel_buffer_size = 1000000 //TODO: Change hardcoded

	self := &MessageMiddleware{
		writers: make([]*WriteWorker, 0, 5), //Initial size
		config:  config,
	}

	err := self.start_connection(config.Address)

	if err != nil {
		return nil, err
	}

	err = self.declare_admin_queue()
	if err != nil {
		return nil, fmt.Errorf("Error trying to declare admin queue: %v", err)
	}

	return self, nil
}

func (self *MessageMiddleware) declare_admin_queue() error {
	_, err := self.queue_declare("admin_control", false)
	if err != nil {
		return nil
	}

	return nil
}

func (self *MessageMiddleware) start_connection(host string) error {
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

func (self *MessageMiddleware) New_queue(config QueueConfig) (chan Message, error) {
	switch config.Direction {
	case "read":
		return self.Read_queue(config)
	case "write":
		return self.Write_queue(config)
	default:
		return nil, fmt.Errorf("Error trying to create a queue: Direction %v is not recognized", config.Direction)
	}
}

func (self *MessageMiddleware) Read_queue(config QueueConfig) (chan Message, error) {
	switch config.Class {
	case "worker":
		return self.create_read_worker_queue(&config)
	case "topic":
		return self.create_read_topic_queue(&config)
	case "fanout":
		return self.create_read_fanout_queue(&config)
	default:
		return nil, fmt.Errorf("Error trying to create a read queue: Class %v is not recognized", config.Class)
	}
}

func (self *MessageMiddleware) create_read_worker_queue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.Channel_buffer_size)
	var q_channel <-chan amqp.Delivery
	var err error

	if len(config.Topic) > 0 {
		if len(config.Source) < 1 {
			return nil, fmt.Errorf("Error trying to create a read worker queue: source not provided")
		}
		q_channel, err = self.initialize_queue_and_exchange(
			config.Source,
			"topic",
			config.Name,
			config.Topic)
		if err != nil {
			return nil, fmt.Errorf("Error trying to create a read worker queue %v: %v", config.Name, err)
		}
	} else {
		if len(config.Name) < 1 {
			return nil, fmt.Errorf("Error trying to create a read worker queue: name not provided")
		}
		//Declare a queue with config.name name
		queue, err := self.queue_declare(config.Name, false)
		if err != nil {
			return nil, Err.Ctx("Couldn't declare the queue", err)
		}
		//Get a channel from the queue
		q_channel, err = self.consume_queue(queue.Name)
		if err != nil {
			return nil, Err.Ctx("Couldn't bind queue to exchange", err)
		}
	}

	//We invoke a new worker, which will listen for new messages on the queue
	go read_worker(q_channel, channel)

	return channel, nil
}

func (self *MessageMiddleware) create_read_topic_queue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.Channel_buffer_size)
	if len(config.Topic) < 1 || len(config.Source) < 1 {
		return nil, fmt.Errorf("Error trying to create a read topic queue: topic or source not provided")
	}
	q_channel, err := self.initialize_queue_and_exchange(
		config.Source,
		"topic",
		config.Name,
		config.Topic)

	if err != nil {
		return nil, fmt.Errorf("Error trying to create a read topic queue %v: %v", config.Source, err)
	}
	//We invoke a new worker, which will listen for new messages on the queue
	go read_worker(q_channel, channel)

	return channel, nil
}

func (self *MessageMiddleware) create_read_fanout_queue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.Channel_buffer_size)
	if len(config.Source) < 1 {
		return nil, fmt.Errorf("Error trying to create a read fanout queue: source not provided")
	}
	q_channel, err := self.initialize_queue_and_exchange(
		config.Source,
		"fanout",
		config.Name,
		"")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a read fanout queue %v: %v", config.Source, err)
	}

	//We invoke a new worker, which will listen for new messages on the queue
	go read_worker(q_channel, channel)

	return channel, nil
}

func read_worker(input <-chan amqp.Delivery, output chan Message) {
	//Will continue until the connection is closed
	for msg := range input {
		messages := decode_string_slice(msg.Body)
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

func (self *MessageMiddleware) initialize_queue_and_exchange(
	exchange_name string,
	exchange_type string,
	queue_name string,
	queue_topic string,
) (<-chan amqp.Delivery, error) {
	//Declare an exchange with the name 'source' and the class 'topic'
	err := self.exchange_declare(exchange_name, exchange_type)
	if err != nil {
		return nil, Err.Ctx("Couldn't declare the exchange", err)
	}
	//Declare a new queue with the name 'name', which could be empty
	queue, err := self.queue_declare(queue_name, false)
	if err != nil {
		return nil, Err.Ctx("Couldn't declare the queue", err)
	}
	//Bind the queue to the exchange
	err = self.queue_bind(queue.Name, queue_topic, exchange_name)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange", err)
	}
	//Bind the queue to the admin "finish" topic
	err = self.queue_bind(queue.Name, "finish", exchange_name)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange, with topic finish", err)
	}
	//Get a channel from the queue
	q_channel, err := self.consume_queue(queue.Name)
	if err != nil {
		return nil, Err.Ctx("Couldn't bind queue to exchange", err)
	}
	return q_channel, nil
}

func (self *MessageMiddleware) Write_queue(config QueueConfig) (chan Message, error) {
	switch config.Class {
	case "worker":
		return self.create_write_worker_queue(&config)
	case "topic":
		return self.create_write_topic_queue(&config)
	case "fanout":
		return self.create_write_fanout_queue(&config)
	default:
		return nil, fmt.Errorf("Error trying to create a write queue: Class %v is not recognized", config.Class)
	}
}

func (self *MessageMiddleware) create_write_worker_queue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.Channel_buffer_size)
	if len(config.Name) < 1 {
		return nil, fmt.Errorf("Error trying to create a write worker queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	_, err := self.queue_declare(config.Name, false)
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write worker queue %v: %v", config.Name, err)
	}

	err = self.notify_new_write_queue_to_admin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	writer := writer_worker(
		config.Name,
		"",
		false,
		config.Name,
		channel,
		self.channel,
		self.config.Notify_on_finish,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) create_write_topic_queue(config *QueueConfig) (chan Message, error) {
	log.Debugf("Creating a work queue %v", config)
	channel := make(chan Message, self.config.Channel_buffer_size)
	if len(config.Name) < 1 {
		return nil, fmt.Errorf("Error trying to create a write topic queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	err := self.exchange_declare(config.Name, "topic")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write topic queue %v: %v", config.Name, err)
	}

	err = self.notify_new_write_queue_to_admin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	writer := writer_worker(
		config.Name,
		config.Name,
		true,
		"",
		channel,
		self.channel,
		self.config.Notify_on_finish,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) create_write_fanout_queue(config *QueueConfig) (chan Message, error) {
	channel := make(chan Message, self.config.Channel_buffer_size)
	if len(config.Name) < 1 {
		return nil, fmt.Errorf("Error trying to create a write fanout queue: name not provided")
	}

	//Declare a new queue with the name 'name', which could be empty
	err := self.exchange_declare(config.Name, "fanout")
	if err != nil {
		return nil, fmt.Errorf("Error trying to create a write fanout queue %v: %v", config.Name, err)
	}

	err = self.notify_new_write_queue_to_admin(config)
	if err != nil {
		return nil, fmt.Errorf("Error notifyng new queue to admin: %v", err)
	}

	writer := writer_worker(
		config.Name,
		config.Name,
		false,
		"",
		channel,
		self.channel,
		self.config.Notify_on_finish,
	)

	self.writers = append(self.writers, writer)

	return channel, nil
}

func (self *MessageMiddleware) exchange_declare(name string, class string) error {
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

func (self *MessageMiddleware) queue_declare(name string, exclusive bool) (amqp.Queue, error) {
	return self.channel.QueueDeclare(
		name,      // name
		false,     // durable
		false,     // delete when unused
		exclusive, // exclusive
		false,     // no-wait
		nil,       // arguments
	)
}

func (self *MessageMiddleware) queue_bind(name string, routing_key string, exchange string) error {
	return self.channel.QueueBind(
		name,        // queue name
		routing_key, // routing key
		exchange,    // exchange
		false,
		nil,
	)
}

func (self *MessageMiddleware) consume_queue(name string) (<-chan amqp.Delivery, error) {
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

func (self *MessageMiddleware) notify_new_write_queue_to_admin(config *QueueConfig) error {

	//TODO: Cambiar el nombre de la variable a algo que tenga más sentido
	//Lo que estamos haciendo acá es no enviar el mensaje si se trata del admin
	if !self.config.Notify_on_finish {
		return nil
	}

	target_queue := "admin_control"
	message := fmt.Sprintf("new_write,%v,%v,%v,%v,%v",
		config.Name,
		config.Class,
		config.Topic,
		config.Source,
		config.Direction)

	err := publish([]string{message}, "", target_queue, self.channel)

	return err
}
