package main

import (
	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func worker_callback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	digestor := NewDigestor()

	lb, err := strconv.ParseUint(envs["load_balance"], 10, 64)
	if err != nil || lb < 1 {
		log.Errorf("load_balance config variable is not valid, must be positive", lb)
		return
	}
	callback := worker.Create_load_balance_callback(
		digestor.filter,
		uint(lb),
		envs["process_group"],
		queues["result"])

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	consumer.Run()
}

func main() {
	// - Start New Worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"load_balance",
			"process_group",
		},
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

	process_worker.Run(worker_callback)
}
