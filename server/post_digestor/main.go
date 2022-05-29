package main

import (
	"fmt"

	"distribuidos/tp2/server/common/consumer"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"

	log "github.com/sirupsen/logrus"
)

// Hacer una funcion o estructura que:

// - Reciba el nombre del proceso en el archivo de configs
// - Reciba el nombre del proceso para imprimir configs
// - Reciba un listado de las envs a parsear
// - Reciba un listado de las colas a levantar

// - Inicie la quit signal
// - Levante las configuraciones, parseando en un hash en lugar de struct
// - Inicie el logger
// - Imprima las configuraciones levantadas
// - Levante el middleware
// - Inicialice las colas y las retorne en un hash (como venimos haciendo)
// - Tenga un método finish que esencialmente llame a finish del mom

// De esa forma cada main solo se ocupará de implementar una mínima lógica de sincronización y de business

func main() {
	// - Definir señal de quit
	quit := worker.Start_quit_signal()
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

	digestor := NewDigestor()

	callback := worker.Create_callback(&config, digestor.filter, queues["result"])

	// - Create and run the consumer
	q := consumer.ConsumerQueues{Input: queues["input"]}
	consumer, err := consumer.New(callback, q, quit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	consumer.Run()
	// digestor.info()
}
