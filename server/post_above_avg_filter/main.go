package main

import (
	"fmt"
	"strconv"

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
		"input_avg",
		"input_post",
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
	log.Info("Starting Post Score Avg Calculator...")

	//- Print config
	worker.Print_config(&config, "Post Score Avg Calculator")

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

	//- Wait for avg result
	avg_result := <-queues["input_avg"]
	log.Debugf("AVG: %v", avg_result.Body)

	avg, err := strconv.ParseFloat(avg_result.Body, 64)
	if err != nil {
		log.Fatalf("Post AVG value %v is not a number: %v", avg_result.Body, err)
	}
	// - Callback definition
	calculator := NewFilter(avg)
	callback := worker.Create_callback(&config, calculator.work, queues["result"])
	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input_post"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}

	consumer.Run()
}
