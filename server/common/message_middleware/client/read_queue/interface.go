package read_queue

import "github.com/streadway/amqp"

type ReadQueue interface {
	Read() <-chan amqp.Delivery
	Close()
}
