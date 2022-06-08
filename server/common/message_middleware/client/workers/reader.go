package message_middleware

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/message_middleware/client/protocol"
	"distribuidos/tp2/server/common/message_middleware/client/read_queue"
	"sync"

	log "github.com/sirupsen/logrus"
)

type ReadWorker struct {
	queue    read_queue.ReadQueue
	output   chan mom.Message
	quit     chan bool
	finished sync.WaitGroup
}

func StartReadWorker(
	queue read_queue.ReadQueue,
	output chan mom.Message,
) *ReadWorker {
	self := &ReadWorker{
		queue:  queue,
		output: output,
		quit:   make(chan bool, 2),
	}

	self.finished.Add(1)
	go self.work()

	return self
}

func (self *ReadWorker) work() {
	input := self.queue.Read()

Loop:
	for {
		select {
		case msg, more := <-input:
			if !more {
				break Loop
			}
			batch := protocol.DecodeStringSlice(msg.Body)
			for _, m := range batch {
				if m == "finish" {
					log.Debugf("Reader has found finish message, closing...")
					close(self.output)
					break Loop
				}
				self.output <- mom.Message{
					Body:  m,
					Topic: msg.RoutingKey,
				}
			}
		case <-self.quit:
			break Loop
		}
	}

	self.queue.Close()
	self.finished.Done()
}

func (self *ReadWorker) Finish() {
	self.quit <- true
	self.finished.Wait()
}
