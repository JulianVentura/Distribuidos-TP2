package main

import (
	"distribuidos/tp2/server/common/worker"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func InitLogger(logLevel string) error {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})
	return nil
}

func main() {

	//TODO: Levantar todo de config
	quit := worker.StartQuitSignal()
	if err := InitLogger("debug"); err != nil {
		fmt.Println("Couldn't initialize logger")
		return
	}

	admin, err := StartAdmin("amqp://rabbitmq", quit)
	if err != nil {
		log.Fatalf("Couldn't start admin", err)
	}
	admin.Run()
}
