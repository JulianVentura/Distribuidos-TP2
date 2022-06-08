package worker

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	mom "distribuidos/tp2/server/common/message_middleware"
	momCli "distribuidos/tp2/server/common/message_middleware/client"
	"distribuidos/tp2/server/common/utils"

	json "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// InitLogger Receives the log level to be set in logrus as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func initLogger(logLevel string) error {
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

func printConfig(config *InitConfig) {
	log.Infof("Configuration %v %v", config.Envs["process_name"], config.Envs["id"])
	for k, v := range config.Envs {
		log.Infof("* %s: %s", k, v)
	}

	log.Infof("* Queues:")

	for qName, qConf := range config.Queues {
		log.Infof(" # %v:", qName)
		log.Infof("  - Name: %v", qConf.Name)
		log.Infof("  - Class: %v", qConf.Class)
		log.Infof("  - Topic: %v", qConf.Topic)
		log.Infof("  - Source: %v", qConf.Source)
		log.Infof("  - Direction: %v", qConf.Direction)
	}
}

func CreateLoadBalanceCallback(
	callback func(string) (string, error),
	loadBalance uint,
	processGroup string,
	resultQueue chan mom.Message) func(string) {

	resultTopic := fmt.Sprintf("%v_result", processGroup)
	return func(msg string) {
		result, err := callback(msg)
		if err == nil {
			BalanceLoadSend(resultQueue, resultTopic, loadBalance, result)
		}
	}
}

func CreateCallback(
	callback func(string) (string, error),
	resultQueue chan mom.Message) func(string) {

	return func(msg string) {
		result, err := callback(msg)
		if err == nil {
			resultQueue <- mom.Message{
				Body: result,
			}
		}
	}
}

func BalanceLoadSend(
	output chan mom.Message,
	topicHeader string,
	processNumber uint,
	message string) {

	parser := utils.NewParser()
	id := parser.ReadN(message, 2)[0]
	target := hash(id) % uint32(processNumber)

	output <- mom.Message{
		Body:  message,
		Topic: fmt.Sprintf("%s.%v", topicHeader, target),
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func StartQuitSignal() chan bool {

	exit := make(chan os.Signal, 1)
	quit := make(chan bool, 1)
	signal.Notify(exit, syscall.SIGTERM)
	go func() {
		<-exit
		quit <- true
	}()

	return quit
}

func parseQueueConfig(data []byte) (mom.QueueConfig, error) {
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

func parseQueuesConfig(data []byte, queues []string) (map[string]mom.QueueConfig, error) {

	queuesConfig, _, _, err := json.Get(data, "queues")
	if err != nil {
		return nil, fmt.Errorf("Config file is bad formatted: %v", err)
	}

	result := make(map[string]mom.QueueConfig)

	for _, q := range queues {
		input_data, _, _, err := json.Get(queuesConfig, q)
		if err != nil {
			return nil, fmt.Errorf("Could not find %v queue in config: %v", q, err)
		}
		queue, err := parseQueueConfig(input_data)
		if err != nil {
			return nil, fmt.Errorf("Queue %v is bad formated: %v", q, err)
		}

		result[q] = queue
	}

	return result, nil
}

func initQueues(msgMiddleware *momCli.MessageMiddleware, config map[string]mom.QueueConfig) (map[string]chan mom.Message, error) {

	queues := make(map[string](chan mom.Message))

	for qName, qConfig := range config {
		q, err := msgMiddleware.NewQueue(qConfig)
		if err != nil {
			return nil, err
		}
		queues[qName] = q
	}

	return queues, nil
}

func expadnQueuesTopic(queues map[string]mom.QueueConfig, id string) {
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
	Mom    *momCli.MessageMiddleware
}

type InitConfig struct {
	Envs   map[string]string
	Queues map[string]mom.QueueConfig
}

func addIfNotExists(list *[]string, val string) {
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
	addIfNotExists(&config.Envs, "id")
	addIfNotExists(&config.Envs, "mom_address")
	addIfNotExists(&config.Envs, "log_level")
	addIfNotExists(&config.Envs, "process_name")
	addIfNotExists(&config.Envs, "process_group")
	addIfNotExists(&config.Envs, "mom_msg_batch_target_size")
	addIfNotExists(&config.Envs, "mom_msg_batch_timeout")
	addIfNotExists(&config.Envs, "mom_channel_buffer_size")
	// - Definir señal de quit
	quit := StartQuitSignal()
	// - Config parse
	cfg, err := initConfig(&config)
	if err != nil {
		return Worker{}, fmt.Errorf("Error on config: %v\n", err)
	}
	//- Logger init
	if err := initLogger(cfg.Envs["log_level"]); err != nil {
		return Worker{}, fmt.Errorf("Couldn't start logger: %v", err)
	}

	//- Middleware config parse
	mom_timeout, err := time.ParseDuration(cfg.Envs["mom_msg_batch_timeout"])
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't parse mom_msg_batch_timeout as time.Duration: %v", err)
	}

	mom_batch_size, err := strconv.ParseUint(cfg.Envs["mom_msg_batch_target_size"], 10, 64)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't parse mom_msg_batch_target_size: %v", err)
	}
	mom_chann_buff_size, err := strconv.ParseUint(cfg.Envs["mom_channel_buffer_size"], 10, 64)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't parse mom_channel_buffer_size: %v", err)
	}

	log.Infof("Starting %v...", cfg.Envs["process_name"])

	//Expand the queues topic with process id
	expadnQueuesTopic(cfg.Queues, cfg.Envs["id"])

	//- Print config
	printConfig(&cfg)

	// - Mom initialization
	mConfig := momCli.MessageMiddlewareConfig{
		Address:                cfg.Envs["mom_address"],
		NotifyAdmin:            true,
		ChannelBufferSize:      uint(mom_chann_buff_size),
		MessageBatchSizeTarget: uint(mom_batch_size),
		MessageBatchTimeout:    mom_timeout,
	}

	messageMiddleware, err := momCli.Start(mConfig)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't start mom: %v", err)
	}

	// - Queues initialization
	queues, err := initQueues(messageMiddleware, cfg.Queues)
	if err != nil {
		return Worker{}, fmt.Errorf("Couldn't connect to mom: %v", err)
	}

	return Worker{
		Config: cfg.Envs,
		Queues: queues,
		Quit:   quit,
		Mom:    messageMiddleware,
	}, nil
}

func (self *Worker) Finish() {
	self.Mom.Finish()
}

func (self *Worker) Run(callback func(map[string]string, map[string]chan mom.Message, chan bool)) {
	callback(self.Config, self.Queues, self.Quit)
}

func initConfig(config *WorkerConfig) (InitConfig, error) {

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
	processGroup := cfg.Envs["process_group"]

	if id == "" || processGroup == "" {
		return cfg, fmt.Errorf("Couldn't parse id or process_group")
	}

	//Parse json config
	path := "./config.json"

	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("Couldn't open file %v. %v\n", path, err)
	}

	//General config
	momAddress, err := json.GetString(jsonData, "general", "mom_address")
	if err != nil {
		return cfg, fmt.Errorf("Couldn't parse mom address from config file")
	}

	cfg.Envs["id"] = id
	cfg.Envs["process_group"] = processGroup
	cfg.Envs["log_level"] = "debug" //Default value
	cfg.Envs["mom_address"] = momAddress

	processData, _, _, err := json.Get(jsonData, processGroup)
	if err != nil {
		return cfg, fmt.Errorf("Config file is bad formatted")
	}

	//Parse process config from json
	err = json.ObjectEach(processData, func(key []byte, value []byte, dataType json.ValueType, offset int) error {
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
	queuesConfig, err := parseQueuesConfig(processData, config.Queues)
	if err != nil {
		return cfg, err
	}

	cfg.Queues = queuesConfig

	//Check if all wanted config keys were found
	for _, k := range config.Envs {
		value, exists := cfg.Envs[k]
		if !exists || len(value) < 1 {
			return cfg, fmt.Errorf("Couldn't find %v config", k)
		}
	}

	return cfg, nil
}
