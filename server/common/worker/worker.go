package worker

import (
	"fmt"
	"io/ioutil"
	"strings"

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

	queues_config, err := utils.Parse_queues_config(process_data, queues_list)
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
