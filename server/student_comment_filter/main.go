package main

import (
	"fmt"
	"strconv"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func workerCallback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	// - Create the business structure and callback
	lb, err := strconv.ParseUint(envs["load_balance"], 10, 64)
	if err != nil || lb < 1 {
		log.Errorf("load_balance config variable is not valid, must be positive: %v", lb)
		return
	}
	filter := NewFilter()
	callback := worker.CreateLoadBalanceCallback(
		filter.filter,
		uint(lb),
		envs["process_group"],
		queues["result"])

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	consumer.Run()
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"process_group",
			"load_balance",
		},
		Queues: []string{
			"input",
			"result",
		},
	}

	processWorker, err := worker.StartWorker(cfg)
	if err != nil {
		fmt.Printf("Error starting new process worker: %v\n", err)
		return
	}
	defer processWorker.Finish()

	// - Run the process worker
	processWorker.Run(workerCallback)
}
