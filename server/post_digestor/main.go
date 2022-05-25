package main

import (
	"fmt"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/utils"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

func create_callback(
	config *worker.WorkerConfiguration,
	callback func(string) (string, error),
	result_q chan mom.Message) func(string) {

	if config.Load_balance > 0 {
		result_topic := fmt.Sprintf("%v_result", config.Process_group)
		return func(msg string) {
			result, err := callback(msg)
			if err == nil {
				utils.Balance_load_send(result_q, result_topic, config.Load_balance, result)
			}
		}
	} else {
		return func(msg string) {
			result, err := callback(msg)
			if err == nil {
				result_q <- mom.Message{
					Body: result,
				}
			}
		}
	}
}

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
		fmt.Printf("Error on config: %v", err)
		return
	}

	//- Logger init
	if err := worker.Init_logger(config.Log_level); err != nil {
		log.Fatalf("%s", err)
	}
	log.Info("Starting Post Digestor...")

	//- Print config
	worker.Print_config(&config, "Post Digestor")

	// - Mom initialization
	msg_middleware, err := mom.Start(config.Mom_address, true)
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
	callback := create_callback(&config, work_callback(), queues["result"])

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	consumer.Run()
}
