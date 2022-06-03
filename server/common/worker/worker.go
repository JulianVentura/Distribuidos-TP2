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

// InitLogger Receives the log level to be set in logrus as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func init_logger(logLevel string) error {
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

func print_config(config *InitConfig) {
	log.Infof("Configuration %v %v", config.Envs["process_name"], config.Envs["id"])
	for k, v := range config.Envs {
		log.Infof("* %s: %s", k, v)
	}

	log.Infof("* Queues:")

	for q_name, q_config := range config.Queues {
		log.Infof(" # %v:", q_name)
		log.Infof("  - Name: %v", q_config.Name)
		log.Infof("  - Class: %v", q_config.Class)
		log.Infof("  - Topic: %v", q_config.Topic)
		log.Infof("  - Source: %v", q_config.Source)
		log.Infof("  - Direction: %v", q_config.Direction)
	}
}

func Create_load_balance_callback(
	callback func(string) (string, error),
	load_balance uint,
	process_group string,
	result_q chan mom.Message) func(string) {

	result_topic := fmt.Sprintf("%v_result", process_group)
	return func(msg string) {
		result, err := callback(msg)
		if err == nil {
			Balance_load_send(result_q, result_topic, load_balance, result)
		}
	}
}

func Create_callback(
	callback func(string) (string, error),
	result_q chan mom.Message) func(string) {

	return func(msg string) {
		result, err := callback(msg)
		if err == nil {
			result_q <- mom.Message{
				Body: result,
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

func parse_queues_config(data []byte, queues []string) (map[string]mom.QueueConfig, error) {

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

func init_queues(msg_middleware *mom.MessageMiddleware, config map[string]mom.QueueConfig) (map[string]chan mom.Message, error) {

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

func expand_queues_topic(queues map[string]mom.QueueConfig, id string) {
	for name, queue := range queues {
		queue.Topic = strings.Replace(queue.Topic, "{id}", id, 1)
		queues[name] = queue
	}
}

type WorkerConfig struct {
	Envs   []string
	Queues []string
}

type Worker struct {
	Config map[string]string
	Queues map[string]chan mom.Message
	Quit   chan bool
	Mom    *mom.MessageMiddleware
}

type InitConfig struct {
	Envs   map[string]string
	Queues map[string]mom.QueueConfig
}

func add_if_not_exists(list *[]string, val string) {
	exists := false
	for _, v := range *list {
		if v == val {
			exists = true
			break
		}
	}

	if !exists {
		*list = append(*list, val)
	}
}

func StartWorker(config WorkerConfig) (Worker, error) {
	// - Adding must have configs
	add_if_not_exists(&config.Envs, "id")
	add_if_not_exists(&config.Envs, "mom_address")
	add_if_not_exists(&config.Envs, "log_level")
	add_if_not_exists(&config.Envs, "process_name")
	add_if_not_exists(&config.Envs, "process_group")
	// - Definir señal de quit
	quit := Start_quit_signal()
	// - Config parse
	cfg, err := init_config(&config)
	if err != nil {
		return Worker{}, fmt.Errorf("Error on config: %v\n", err)
	}
	//- Logger init
	if err := init_logger(cfg.Envs["log_level"]); err != nil {
		return Worker{}, fmt.Errorf("Couldn't start logger: %v", err)
	}
	log.Infof("Starting %v...", cfg.Envs["process_name"])

	//Expand the queues topic with process id
	expand_queues_topic(cfg.Queues, cfg.Envs["id"])

	//- Print config
	print_config(&cfg)

	// - Mom initialization
	m_config := mom.MessageMiddlewareConfig{
		Address:          cfg.Envs["mom_address"],
		Notify_on_finish: true,
	}

	msg_middleware, err := mom.Start(m_config)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't start mom: %v", err)
	}

	// - Queues initialization
	queues, err := init_queues(msg_middleware, cfg.Queues)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't connect to mom: %v", err)
	}

	return Worker{
		Config: cfg.Envs,
		Queues: queues,
		Quit:   quit,
		Mom:    msg_middleware,
	}, nil
}

func (self *Worker) Finish() {
	self.Mom.Finish()
}

func (self *Worker) Run(callback func(map[string]string, map[string]chan mom.Message, chan bool)) {
	callback(self.Config, self.Queues, self.Quit)
}

func init_config(config *WorkerConfig) (InitConfig, error) {

	cfg := InitConfig{
		Envs: make(map[string]string),
	}

	v := viper.New()

	v.AutomaticEnv()
	// v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Add env variables supported
	for _, env := range config.Envs {
		v.BindEnv(strings.Split(env, "_")...)
		cfg.Envs[env] = v.GetString(env)
	}

	id := cfg.Envs["id"]
	process_group := cfg.Envs["process_group"]

	if len(id) < 1 || len(process_group) < 1 {
		return cfg, fmt.Errorf("Couldn't parse id or process_group")
	}

	//Parse json config
	path := "./config.json"

	json_data, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("Couldn't open file %v. %v\n", path, err)
	}

	//General config
	mom_address, err := json.GetString(json_data, "general", "mom_address")
	if err != nil {
		return cfg, fmt.Errorf("Couldn't parse mom address from config file")
	}

	cfg.Envs["id"] = id
	cfg.Envs["process_group"] = process_group
	cfg.Envs["log_level"] = "debug" //Default value
	cfg.Envs["mom_address"] = mom_address

	process_data, _, _, err := json.Get(json_data, process_group)
	if err != nil {
		return cfg, fmt.Errorf("Config file is bad formatted")
	}

	//Parse process config from json
	err = json.ObjectEach(process_data, func(key []byte, value []byte, dataType json.ValueType, offset int) error {
		if string(key) == "queues" {
			return nil
		}
		cfg.Envs[string(key)] = string(value)
		return nil
	})

	if err != nil {
		return cfg, err
	}

	//Parse queues config from json
	queues_config, err := parse_queues_config(process_data, config.Queues)
	if err != nil {
		return cfg, err
	}

	cfg.Queues = queues_config

	//Check if all wanted config keys were found
	for _, k := range config.Envs {
		value, exists := cfg.Envs[k]
		if !exists || len(value) < 1 {
			return cfg, fmt.Errorf("Couldn't find %v config", k)
		}
	}

	return cfg, nil
}
