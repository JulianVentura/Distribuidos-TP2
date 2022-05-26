package consumer

import (
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
)

type ConsumerQueues struct {
	Input chan mom.Message
}

type Consumer struct {
	queues          ConsumerQueues
	on_msg_callback func(string)
	quit            chan bool
	has_finished    chan bool
}

func New(
	on_msg_callback func(string),
	queues ConsumerQueues,
	quit chan bool,
) (*Consumer, error) {

	self := &Consumer{
		queues:          queues,
		on_msg_callback: on_msg_callback,
		quit:            quit,
		has_finished:    make(chan bool, 2),
	}

	return self, nil
}

func (self *Consumer) Finish() {
	self.quit <- true
	<-self.has_finished
}

func (self *Consumer) finish() {
	self.has_finished <- true
}

func (self *Consumer) Run() {
	defer self.finish()

Loop:
	for {
		select {
		case <-self.quit:
			break Loop
		case m, open := <-self.queues.Input:
			if !open {
				break Loop
			}
			self.on_msg_callback(m.Body)
		}
	}
}
