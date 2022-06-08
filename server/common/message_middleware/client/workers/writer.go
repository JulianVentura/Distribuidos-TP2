package message_middleware

import (
	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/message_middleware/client/protocol"
	wq "distribuidos/tp2/server/common/message_middleware/client/write_queue"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type WriteWorkerConfig struct {
	MessageBatchSizeTarget uint
	MessageBatchTimeout    time.Duration
}

type WriteWorker struct {
	config     WriteWorkerConfig
	input      chan mom.Message
	queue      wq.WriteQueue
	batchTable BatchTable
	quit       chan bool
	finished   sync.WaitGroup
}

func StartWriteWorker(
	config WriteWorkerConfig,
	input chan mom.Message,
	queue wq.WriteQueue,
) *WriteWorker {

	self := &WriteWorker{
		config: config,
		input:  input,
		queue:  queue,
		quit:   make(chan bool, 2),
	}

	self.batchTable = createBatchTable(self.sendCallback, config.MessageBatchSizeTarget)
	self.finished.Add(1)

	go self.run()

	return self
}

func (self *WriteWorker) sendCallback(topic string, messages []string) {
	encoded := protocol.EncodeStringSlice(messages)
	if err := self.queue.Write(encoded, topic); err != nil {
		log.Errorf("Error sending message on writer")
		return
	}
}

func (self *WriteWorker) Finish() {
	self.quit <- true
	self.finished.Wait()
}

func (self *WriteWorker) run() {
	var timeout <-chan time.Time
Loop:
	for {
		select {
		case <-timeout:
			self.batchTable.flush()
		case msg, more := <-self.input:
			if self.batchTable.isEmpty() {
				timeout = time.After(self.config.MessageBatchTimeout)
			}
			if !more {
				break Loop
			}
			self.batchTable.addEntry(msg.Topic, msg.Body)
		case <-self.quit:
			self.sendLastMessages()
			break Loop
		}
	}

	self.batchTable.flush()
	self.queue.Close()
	self.finished.Done()
}

func (self *WriteWorker) sendLastMessages() {

Loop:
	for {
		select {
		case msg := <-self.input:
			self.batchTable.addEntry(msg.Topic, msg.Body)
		default:
			break Loop
		}
	}
}
