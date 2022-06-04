package message_middleware

import (
	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

func publish(messages []string, exchange string, routing_key string, output *amqp.Channel) error {
	encoded := encode_string_slice(messages)
	err := output.Publish(
		exchange,    // exchange
		routing_key, // routing key
		false,       // mandatory
		false,
		amqp.Publishing{
			Body: encoded,
		})

	if err != nil {
		log.Errorf("Error trying to publish a message to mom: %v", err)
		return err
	}

	return nil
}
