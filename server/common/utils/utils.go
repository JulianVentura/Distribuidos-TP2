package utils

import (
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"fmt"
	"hash/fnv"
	"os"
	"os/signal"
	"strings"
	"syscall"

	json "github.com/buger/jsonparser"
)

func Balance_load_send(
	output chan mom.Message,
	topic_header string,
	process_number uint,
	message string) {

	id := strings.SplitN(message, ",", 2)[0]
	target := hash(id) % uint32(process_number)

	output <- mom.Message{
		Body:  message,
		Topic: fmt.Sprintf("%s.%v", topic_header, target),
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func Start_quit_signal() chan bool {

	exit := make(chan os.Signal, 1)
	quit := make(chan bool, 1)
	signal.Notify(exit, syscall.SIGTERM)
	go func() {
		<-exit
		quit <- true
	}()

	return quit
}

func parse_queue_config(data []byte) (mom.QueueConfig, error) {
	config := mom.QueueConfig{}
	name, err := json.GetString(data, "name")
	if err != nil {
		name = ""
	}
	class, err := json.GetString(data, "class")
	if err != nil {
		return config, err
	}
	topic, err := json.GetString(data, "topic")
	if err != nil {
		topic = ""
	}
	source, err := json.GetString(data, "source")
	if err != nil {
		source = ""
	}
	direction, err := json.GetString(data, "direction")
	if err != nil {
		return config, err
	}

	config.Name = name
	config.Class = class
	config.Topic = topic
	config.Source = source
	config.Direction = direction

	return config, nil
}

func Parse_queues_config(data []byte, queues []string) (map[string]mom.QueueConfig, error) {

	queues_config, _, _, err := json.Get(data, "queues")
	if err != nil {
		return nil, fmt.Errorf("Config file is bad formatted: %v", err)
	}

	result := make(map[string]mom.QueueConfig)

	for _, q := range queues {
		input_data, _, _, err := json.Get(queues_config, q)
		if err != nil {
			return nil, fmt.Errorf("Could not find %v queue in config: %v", q, err)
		}
		queue, err := parse_queue_config(input_data)
		if err != nil {
			return nil, fmt.Errorf("Queue %v is bad formated: %v", q, err)
		}

		result[q] = queue
	}

	return result, nil
}

func Init_queues(msg_middleware *mom.MessageMiddleware, config map[string]mom.QueueConfig) (map[string]chan mom.Message, error) {

	queues := make(map[string](chan mom.Message))

	for q_name, q_config := range config {
		q, err := msg_middleware.New_queue(q_config)
		if err != nil {
			return nil, err
		}
		queues[q_name] = q
	}

	return queues, nil
}

func Expand_queues_topic(queues map[string]mom.QueueConfig, id uint) {
	id_s := fmt.Sprintf("%v", id)
	for name, queue := range queues {
		queue.Topic = strings.Replace(queue.Topic, "{id}", id_s, 1)
		queues[name] = queue
	}
}
