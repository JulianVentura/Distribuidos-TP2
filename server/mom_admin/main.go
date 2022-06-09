package main

import (
	"distribuidos/tp2/server/common/utils"
	"fmt"
	"io/ioutil"

	json "github.com/buger/jsonparser"

	log "github.com/sirupsen/logrus"
)

type MomAdminConfig struct {
	momAddress string
	logLevel   string
}

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

func InitConfig(path string) (MomAdminConfig, error) {
	config := MomAdminConfig{}
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("Couldn't open file %v. %v\n", path, err)
	}

	//General config
	momAddress, err := json.GetString(jsonData, "general", "mom_address")
	if err != nil {
		return config, fmt.Errorf("Couldn't parse mom address from config file")
	}
	//Admin Config
	logLevel, err := json.GetString(jsonData, "mom_admin", "log_level")
	if err != nil {
		logLevel = "info" //Default
	}

	config.logLevel = logLevel
	config.momAddress = momAddress

	return config, nil
}

func main() {

	quit := utils.StartQuitSignal()
	config, err := InitConfig("./config.json")
	if err != nil {
		fmt.Printf("Couldn't parse configuration: %v", err)
	}
	if err := InitLogger(config.logLevel); err != nil {
		fmt.Println("Couldn't initialize logger")
		return
	}

	admin, err := StartAdmin(config.momAddress, quit)
	if err != nil {
		log.Fatalf("Couldn't start admin", err)
	}
	admin.Run()
}
