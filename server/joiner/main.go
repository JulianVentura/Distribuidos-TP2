package main

import (
	"fmt"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func main() {
	// - Definir señal de quit
	quit := worker.Start_quit_signal()
	// - Definimos lista de colas
	queues_list := []string{
		"base_input",
		"join_input",
		"result",
	}

	// - Config parse
	config, err := worker.Init_config(queues_list)
	if err != nil {
		fmt.Printf("Error on config: %v\n", err)
		return
	}

	//- Logger init
	if err := worker.Init_logger(config.Log_level); err != nil {
		log.Fatalf("%s", err)
	}
	log.Infof("Starting %v...", config.Process_group)

	//Expand the queues topic with process id
	worker.Expand_queues_topic(config.Queues, config.Id)

	//- Print config
	worker.Print_config(&config, config.Process_group)

	// - Mom initialization
	m_config := mom.MessageMiddlewareConfig{
		Address:          config.Mom_address,
		Notify_on_finish: true,
	}

	msg_middleware, err := mom.Start(m_config)
	if err != nil {
		log.Fatalf("Couldn't connect to mom: %v", err)
	}

	defer msg_middleware.Finish()
	// - Queues initialization

	queues, err := worker.Init_queues(msg_middleware, config.Queues)
	if err != nil {
		log.Fatalf("Couldn't connect to mom: %v", err)
	}
	// - Callback definition
	joiner := NewJoiner()

	// - Create and run the consumer for the post input
	q_p := consumer.ConsumerQueues{Input: queues["base_input"]}
	post_consumer, err := consumer.New(joiner.Add, q_p, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	post_consumer.Run()

	result := queues["result"]
	callback := func(input string) {
		r, err := joiner.Join(input)
		if err == nil {
			result <- mom.Message{
				Body: r,
			}
		}
	}

	// - Create and run the consumer for the join input
	q_s := consumer.ConsumerQueues{Input: queues["join_input"]}
	sentiment_consumer, err := consumer.New(callback, q_s, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	sentiment_consumer.Run()
	joiner.info()
}
