package main

import (
	"fmt"
	"strconv"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func workerCallback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	lb, err := strconv.ParseUint(envs["load_balance"], 10, 64)
	if err != nil || lb < 1 {
		log.Errorf("load_balance config variable is not valid, must be positive", lb)
		return
	}
	//- Wait for avg result
	avgResult, valid := <-queues["input_avg"]
	if !valid {
		//Close received
		return
	}

	avg, err := strconv.ParseFloat(string(avgResult.Body), 64)
	if err != nil {
		log.Errorf("Post AVG value %v is not a number: %v", string(avgResult.Body), err)
		return
	}
	// - Business structure initialization
	calculator := NewFilter(avg)
	callback := worker.CreateLoadBalanceCallback(
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

	processWorker, err := worker.StartWorker(cfg)
	if err != nil {
		fmt.Printf("Error starting new process worker: %v\n", err)
		return
	}
	defer processWorker.Finish()

	// - Run the process worker
	processWorker.Run(workerCallback)
}
