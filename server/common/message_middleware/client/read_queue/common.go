package read_queue

import (
	"distribuidos/tp2/server/common/message_middleware/client/util"
	"fmt"

	"github.com/streadway/amqp"
)

func initializeQueueAndExchange(
	channel *amqp.Channel,
	exchangeName string,
	exchangeType string,
	queueName string,
	queueTopic string,
	queueExclusive bool,
) (<-chan amqp.Delivery, error) {
	//Declare an exchange with the name 'source' and the class 'topic'
	err := util.ExchangeDeclare(channel, exchangeName, exchangeType)
	if err != nil {
		return nil, fmt.Errorf("Couldn't declare the exchange: %v", err)
	}
	//Declare a new queue with the name 'name', which could be empty
	queue, err := util.QueueDeclare(channel, queueName, queueExclusive)
	if err != nil {
		return nil, fmt.Errorf("Couldn't declare the queue: %v", err)
	}
	//Bind the queue to the exchange
	err = util.QueueBind(channel, queue.Name, queueTopic, exchangeName)
	if err != nil {
		return nil, fmt.Errorf("Couldn't bind queue to exchange: %v", err)
	}
	//Bind the queue to the admin "finish" topic
	err = util.QueueBind(channel, queue.Name, "finish", exchangeName)
	if err != nil {
		return nil, fmt.Errorf("Couldn't bind queue to exchange, with topic finish: %v", err)
	}
	//Get a channel from the queue
	amqpChann, err := util.ConsumeQueue(channel, queue.Name)
	if err != nil {
		return nil, fmt.Errorf("Couldn't bind queue to exchange: %v", err)
	}
	return amqpChann, nil
}
