package worker

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/utils"

	json "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type WorkerConfiguration struct {
	Id            uint
	Process_group string
	Mom_address   string
	Load_balance  uint
	Log_level     string
	Queues        map[string]mom.QueueConfig
}

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func Init_config(queues_list []string) (WorkerConfiguration, error) {

	config := WorkerConfiguration{}

	v := viper.New()

	v.AutomaticEnv()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Add env variables supported
	v.BindEnv("id")
	v.BindEnv("load", "balance")
	v.BindEnv("process", "group")

	config.Id = v.GetUint("id")
	config.Load_balance = v.GetUint("load.balance")
	config.Process_group = v.GetString("process.group")

	//Parse json config

	path := "./config.json" //TODO: Pasar a env?

	json_data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("Couldn't open file %v. %v\n", path, err)
	}

	//General config
	mom_address, err := json.GetString(json_data, "general", "mom_address")
	if err != nil {
		return config, fmt.Errorf("Couldn't parse mom address from config file")
	}

	//Process config
	process_data, _, _, err := json.Get(json_data, config.Process_group)
	if err != nil {
		return config, fmt.Errorf("Config file is bad formatted")
	}

	log_level, err := json.GetString(json_data, config.Process_group, "log_level")
	if err != nil {
		log_level = "debug"
	}

	queues_config, err := Parse_queues_config(process_data, queues_list)
	if err != nil {
		return config, err
	}

	//We know that those queues where found, so this won't fail

	config.Queues = queues_config
	config.Log_level = log_level
	config.Mom_address = mom_address

	return config, nil

}

// InitLogger Receives the log level to be set in logrus as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func Init_logger(logLevel string) error {
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

func Print_config(config *WorkerConfiguration, process_name string) {
	log.Infof("%v %v Configuration", process_name, config.Id)
	log.Infof("Process group: %s", config.Process_group)
	log.Infof("Mom Address: %s", config.Mom_address)
	log.Infof("Log Level: %s", config.Log_level)
	log.Infof("Load Balance: %v", config.Load_balance)
	log.Infof("Queues:")

	for q_name, q_config := range config.Queues {
		log.Infof(" # %v:", q_name)
		log.Infof("  - Name: %v", q_config.Name)
		log.Infof("  - Class: %v", q_config.Class)
		log.Infof("  - Topic: %v", q_config.Topic)
		log.Infof("  - Source: %v", q_config.Source)
		log.Infof("  - Direction: %v", q_config.Direction)
	}
}

func Notify_finish(config *WorkerConfiguration, control_q chan mom.Message) {
	log.Infof("Se envía fin de %v.%v", config.Process_group, config.Id)
	control_q <- mom.Message{
		Topic: fmt.Sprintf("stage_finished.%v.%v", config.Process_group, config.Id),
	}
}

func Create_callback(
	config *WorkerConfiguration,
	callback func(string) (string, error),
	result_q chan mom.Message) func(string) {

	if config.Load_balance > 0 {
		result_topic := fmt.Sprintf("%v_result", config.Process_group)
		return func(msg string) {
			result, err := callback(msg)
			if err == nil {
				Balance_load_send(result_q, result_topic, config.Load_balance, result)
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

func Balance_load_send(
	output chan mom.Message,
	topic_header string,
	process_number uint,
	message string) {
	parser := utils.NewParser()
	id := parser.ReadN(message, 2)[0]
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
