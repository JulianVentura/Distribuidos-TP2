package main

import (
	"fmt"
	"strconv"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func parseConfigs(envs map[string]string) (JoinerConfig, error) {
	config := JoinerConfig{}
	keepId, err := strconv.ParseBool(envs["keep_id"])
	if err != nil {
		return config, fmt.Errorf("Couldn't parse keep_id: %v", err)
	}
	baseBodySize, err := strconv.ParseUint(envs["base_body_size"], 10, 64)
	if err != nil {
		return config, fmt.Errorf("Couldn't parse base_body_size: %v", err)
	}
	toJoinBodySize, err := strconv.ParseUint(envs["to_join_body_size"], 10, 64)
	if err != nil {
		return config, fmt.Errorf("Couldn't parse to_join_body_size: %v", err)
	}
	filterDuplicates, err := strconv.ParseBool(envs["filter_duplicates"])
	if err != nil {
		return config, fmt.Errorf("Couldn't parse filter_duplicates: %v", err)
	}
	config.keepId = keepId
	config.baseBodySize = uint(baseBodySize)
	config.toJoinBodySize = uint(toJoinBodySize)
	config.filterDuplicates = filterDuplicates
	return config, nil
}

func workerCallback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	// - Parse the env configs
	config, err := parseConfigs(envs)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	// - Create the business structure
	joiner := NewJoiner(config)

	// - Create and run the consumer for the post input
	baseConsumerQueues := consumer.ConsumerQueues{Input: queues["base_input"]}
	baseConsumer, err := consumer.New(joiner.Add, baseConsumerQueues, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	baseConsumer.Run()

	// - Create and run the consumer for the join input
	callback := worker.CreateCallback(joiner.Join, queues["result"])
	joinConsumerQueues := consumer.ConsumerQueues{Input: queues["join_input"]}
	joinConsumer, err := consumer.New(callback, joinConsumerQueues, quit)
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	joinConsumer.Run()
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"keep_id",
			"base_body_size",
			"to_join_body_size",
			"filter_duplicates",
		},
		Queues: []string{
			"base_input",
			"join_input",
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
