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
	//- Wait for avg result
	avg_result := <-queues["input_avg"]
	log.Debugf("AVG: %v", avg_result.Body)

	avg, err := strconv.ParseFloat(avg_result.Body, 64)
	if err != nil {
		log.Errorf("Post AVG value %v is not a number: %v", avg_result.Body, err)
		return
	}
	// - Business structure initialization
	calculator := NewFilter(avg)
	callback := worker.Create_load_balance_callback(
		calculator.work,
		uint(lb),
		envs["process_group"],
		queues["result"])
	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input_post"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	consumer.Run()
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"load_balance",
			"process_group",
		},
		Queues: []string{
			"input_avg",
			"input_post",
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
