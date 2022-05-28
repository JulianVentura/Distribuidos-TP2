package main

import (
	"fmt"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/utils"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func main() {
	// - Definir señal de quit
	quit := utils.Start_quit_signal()
	// - Definimos lista de colas
	queues_list := []string{
		"input",
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
	queues, err := utils.Init_queues(msg_middleware, config.Queues)
	if err != nil {
		log.Fatalf("Couldn't connect to mom: %v", err)
	}
	// - Callback definition
	calculator := NewCalculator()

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(calculator.add, q, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	consumer.Run()

	result := calculator.get_result()
	log.Infof("AVG: %v", result)

	//- Send the result into result queue
	queues["result"] <- mom.Message{
		Body: calculator.get_result(),
	}
}
