package main

import (
	"distribuidos/tp2/server/admin/src/admin"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"io/ioutil"

	"github.com/spf13/viper"

	json "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

// InitLogger Receives the log level to be set in logrus as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
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

func print_config(config *admin.AdminConfig, q_config map[string]mom.QueueConfig) {
	log.Infof("Admin Configuration")
	log.Infof("Server Address: %s", config.Server_address)
	log.Infof("Mom Address: %s", config.Mom_address)
	log.Infof("Log Level: %s", config.Log_level)
	log.Infof("Queues:")

	for q_name, q_config := range q_config {
		log.Infof(" # %v:", q_name)
		log.Infof("  - Name: %v", q_config.Name)
		log.Infof("  - Class: %v", q_config.Class)
		log.Infof("  - Topic: %v", q_config.Topic)
		log.Infof("  - Source: %v", q_config.Source)
		log.Infof("  - Direction: %v", q_config.Direction)
	}

}
func init_config() (admin.AdminConfig, map[string]mom.QueueConfig, error) {

	config := admin.AdminConfig{}

	v := viper.New()

	v.AutomaticEnv()

	//Parse json config

	path := "./config.json" //TODO: Pasar a env?

	json_data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, nil, fmt.Errorf("Couldn't open file %v. %v\n", path, err)
	}

	//General config
	mom_address, err := json.GetString(json_data, "general", "mom_address")
	if err != nil {
		return config, nil, fmt.Errorf("Couldn't parse mom address from config file")
	}
	//Process config
	admin_data, _, _, err := json.Get(json_data, "admin")
	if err != nil {
		return config, nil, fmt.Errorf("Config file is bad formatted")
	}

	server_address, err := json.GetString(json_data, "admin", "server_address")
	if err != nil {
		return config, nil, fmt.Errorf("Config file is bad formatted")
	}

	log_level, err := json.GetString(json_data, "admin", "log_level")
	if err != nil {
		log_level = "debug"
	}

	queues_list := []string{
		"posts",
		"comments",
		"average_result",
		"best_sent_meme_result",
	}

	queues_config, err := utils.Parse_queues_config(admin_data, queues_list)
	if err != nil {
		return config, nil, err
	}

	//We know that those queues where found, so this won't fail

	config.Log_level = log_level
	config.Mom_address = mom_address
	config.Server_address = server_address

	return config, queues_config, nil

}
func main() {

	config, q_configs, err := init_config()
	if err != nil {
		fmt.Printf("Error found on configuration. %v\n", err)
		return
	}

	if err := InitLogger(config.Log_level); err != nil {
		log.Fatalf("%s", err)
	}

	log.Info("Starting Server Admin...")

	print_config(&config, q_configs)

	// - Create quit signal handler
	quit := utils.Start_quit_signal()

	// - Mom initialization
	m_config := mom.MessageMiddlewareConfig{
		Address:          "amqp://rabbitmq",
		Notify_on_finish: true,
	}

	msg_middleware, err := mom.Start(m_config)
	if err != nil {
		log.Fatalf("Couldn't connect to mom: %v", err)
	}

	defer msg_middleware.Finish()

	queues, err := utils.Init_queues(msg_middleware, q_configs)
	if err != nil {
		log.Fatalf("Couldn't init queues: %v", err)
	}

	admin_q := admin.AdminQueues{
		Posts:          queues["posts"],
		Comments:       queues["comments"],
		Average_result: queues["average_result"],
		Best_meme:      queues["best_sent_meme_result"],
	}
	admin, err := admin.New(config, admin_q, quit)
	if err != nil {
		log.Fatalf("Error starting server. %v", err)
		return
	}
	admin.Run()

	log.Infof("Server Admin has been finished...")
}
