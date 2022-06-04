package main

import (
	"fmt"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func worker_callback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	// - Create the business structure
	downloader := NewDownloader()

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(downloader.work, q, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	consumer.Run()

	result := downloader.get_result()
	log.Infof("Meme of size %v", len(result))
	//- Send the result into result queue
	queues["result"] <- mom.Message{
		Body: result,
	}
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{},
		Queues: []string{
			"input",
			"result",
		},
	}

	process_worker, err := worker.StartWorker(cfg)
	if err != nil {
		fmt.Printf("Error starting new process worker: %v\n", err)
		return
	}
	defer process_worker.Finish()

	// - Run the process worker
	process_worker.Run(worker_callback)
}
