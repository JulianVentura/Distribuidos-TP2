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
	calculator := NewCalculator()

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(calculator.add, q, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	consumer.Run()

	result := calculator.get_result()
	log.Infof("AVG: %v", result)

	//- Send the result into result queue
	queues["result"] <- mom.Message{
		Body: calculator.get_result(),
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
