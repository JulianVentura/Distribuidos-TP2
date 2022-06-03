package main

import (
	"fmt"
	"strconv"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func parse_configs(envs map[string]string) (JoinerConfig, error) {
	config := JoinerConfig{}
	keep_id, err := strconv.ParseBool(envs["keep_id"])
	if err != nil {
		return config, fmt.Errorf("Couldn't parse keep_id: %v", err)
	}
	base_body_size, err := strconv.ParseUint(envs["base_body_size"], 10, 64)
	if err != nil {
		return config, fmt.Errorf("Couldn't parse base_body_size: %v", err)
	}
	to_join_body_size, err := strconv.ParseUint(envs["to_join_body_size"], 10, 64)
	if err != nil {
		return config, fmt.Errorf("Couldn't parse to_join_body_size: %v", err)
	}
	config.keep_id = keep_id
	config.base_body_size = uint(base_body_size)
	config.to_join_body_size = uint(to_join_body_size)
	return config, nil
}

func worker_callback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	// - Parse the env configs
	config, err := parse_configs(envs)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	// - Create the business structure
	joiner := NewJoiner(config)

	// - Create and run the consumer for the post input
	q_p := consumer.ConsumerQueues{Input: queues["base_input"]}
	post_consumer, err := consumer.New(joiner.Add, q_p, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	post_consumer.Run()

	// - Create and run the consumer for the join input
	callback := worker.Create_callback(joiner.Join, queues["result"])
	q_s := consumer.ConsumerQueues{Input: queues["join_input"]}
	sentiment_consumer, err := consumer.New(callback, q_s, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	sentiment_consumer.Run()
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"keep_id",
			"base_body_size",
			"to_join_body_size",
		},
		Queues: []string{
			"base_input",
			"join_input",
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
