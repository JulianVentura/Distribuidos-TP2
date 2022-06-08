package consumer

import (
	mom "distribuidos/tp2/server/common/message_middleware"
)

type ConsumerQueues struct {
	Input chan mom.Message
}

type Consumer struct {
	queues        ConsumerQueues
	onMsgCallback func(string)
	quit          chan bool
	hasFinished   chan bool
}

func New(
	onMsgCallback func(string),
	queues ConsumerQueues,
	quit chan bool,
) (*Consumer, error) {

	self := &Consumer{
		queues:        queues,
		onMsgCallback: onMsgCallback,
		quit:          quit,
		hasFinished:   make(chan bool, 2),
	}

	return self, nil
}

func (self *Consumer) Finish() {
	self.quit <- true
	<-self.hasFinished
}

func (self *Consumer) finish() {
	self.hasFinished <- true
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
			self.onMsgCallback(m.Body)
		}
	}
}
