package util

import (
	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

func ExchangeDeclare(
	chann *amqp.Channel,
	name string,
	class string) error {

	return chann.ExchangeDeclare(
		name,  // name
		class, // type
		false, // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

func QueueDeclare(
	chann *amqp.Channel,
	name string,
	exclusive bool) (amqp.Queue, error) {

	return chann.QueueDeclare(
		name,      // name
		false,     // durable
		false,     // delete when unused
		exclusive, // exclusive
		false,     // no-wait
		nil,       // arguments
	)
}

func QueueBind(
	chann *amqp.Channel,
	name string,
	routingKey string,
	exchange string) error {

	return chann.QueueBind(
		name,       // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
}

func ConsumeQueue(
	chann *amqp.Channel,
	name string) (<-chan amqp.Delivery, error) {

	return chann.Consume(
		name,       // queue
		"consumer", // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
}

func Publish(message []byte, exchange string, routingKey string, output *amqp.Channel) error {
	err := output.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,
		amqp.Publishing{
			Body: message,
		})

	if err != nil {
		log.Errorf("Error trying to publish a message to mom: %v", err)
		return err
	}

	return nil
}
