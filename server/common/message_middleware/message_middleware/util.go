package message_middleware

import (
	log "github.com/sirupsen/logrus"
	amqp "github.com/streadway/amqp"
)

func publish(messages []string, exchange string, routingKey string, output *amqp.Channel) error {
	encoded := encodeStringSlice(messages)
	err := output.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
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
