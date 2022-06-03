package main

import (
	"fmt"
	"strconv"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func worker_callback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	lb, err := strconv.ParseUint(envs["load_balance"], 10, 64)
	if err != nil || lb < 1 {
		log.Errorf("load_balance config variable is not valid, must be positive", lb)
		return
	}
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

	//- Send the result into result queue
	result := calculator.get_result()

	out := queues["result"]
	result_topic := fmt.Sprintf("%v_result", envs["process_group"])

	for _, r := range result {
		worker.Balance_load_send(out, result_topic, uint(lb), r)
	}
}

func main() {
	// - Create a new process worker
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

	// - Run the process worker
	process_worker.Run(worker_callback)
}
