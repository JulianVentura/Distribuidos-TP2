package main

import (
	"distribuidos/tp2/server/admin/src/admin"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/worker"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func worker_callback(envs map[string]string, queues map[string]chan mom.Message, quit chan bool) {
	admin_q := admin.AdminQueues{
		Posts:          queues["posts"],
		Comments:       queues["comments"],
		Average_result: queues["average_result"],
		Best_meme:      queues["best_sent_meme_result"],
		School_memes:   queues["best_school_memes_result"],
	}
	admin, err := admin.New(envs["server_address"], admin_q, quit)
	if err != nil {
		log.Fatalf("Error starting server. %v", err)
		return
	}
	admin.Run()

	log.Infof("Server Admin has been finished...")
}

func main() {
	// - Create a new process worker
	cfg := worker.WorkerConfig{
		Envs: []string{
			"server_address",
		},
		Queues: []string{
			"posts",
			"comments",
			"average_result",
			"best_sent_meme_result",
			"best_school_memes_result",
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
