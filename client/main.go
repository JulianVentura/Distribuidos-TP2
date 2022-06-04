package main

import (
	"distribuidos/tp2/client/client"
	Err "distribuidos/tp2/common/errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func InitConfig() (client.ClientConfig, error) {
	v := viper.New()
	errResult := client.ClientConfig{}
	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported
	v.BindEnv("id")
	v.BindEnv("server", "address")
	v.BindEnv("loop-period", "post")
	v.BindEnv("loop-period", "comment")
	v.BindEnv("files-path", "post")
	v.BindEnv("files-path", "comment")
	v.BindEnv("log", "level")

	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("Configuration could not be read from config file. Using env variables instead")
	}

	if _, err := time.ParseDuration(v.GetString("loop-period.comment")); err != nil {
		return errResult, Err.Ctx("Could not parse CLI_LOOP-PERIOD_COMMENT env var as time.Duration.", err)
	}

	if _, err := time.ParseDuration(v.GetString("loop-period.post")); err != nil {
		return errResult, Err.Ctx("Could not parse CLI_LOOP-PERIOD_POST env var as time.Duration.", err)
	}

	c_config := client.ClientConfig{
		Id:                    v.GetUint("id"),
		LogLevel:              v.GetString("log.level"),
		ServerAddress:         v.GetString("server.address"),
		LoopPeriodPost:        v.GetDuration("loop-period.post"),
		LoopPeriodComment:     v.GetDuration("loop-period.comment"),
		FilePathPost:          v.GetString("files-path.post"),
		FilePathComment:       v.GetString("files-path.comment"),
		FilePathSentimentMeme: v.GetString("files-path.sentiment-meme"),
		FilePathSchoolMemes:   v.GetString("files-path.school-memes"),
	}

	return c_config, nil
}

func PrintConfig(config *client.ClientConfig) {
	log.Printf("Client %v configuration: \n", config.Id)
	log.Printf(" - ID: %v\n", config.Id)
	log.Printf(" - Server Address: %v\n", config.ServerAddress)
	log.Printf(" - Loop Period Post: %v\n", config.LoopPeriodPost)
	log.Printf(" - Loop Period Comment: %v\n", config.LoopPeriodComment)
	log.Printf(" - File Path Post: %v\n", config.FilePathPost)
	log.Printf(" - File Path Comment: %v\n", config.FilePathComment)
	log.Printf(" - File Path Sentiment Meme: %v\n", config.FilePathSentimentMeme)
	log.Printf(" - File Path School Memes: %v\n", config.FilePathSchoolMemes)
	log.Print("----\n\n")
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

func main() {
	fmt.Println("Starting Client...")
	config, err := InitConfig()
	if err != nil {
		fmt.Printf("Error found on configuration. %v\n", err)
		return
	}

	if err := InitLogger(config.LogLevel); err != nil {
		log.Fatalf("%s", err)
	}
	PrintConfig(&config)

	cli, err := client.Start(config)

	if err != nil {
		log.Fatalf("Error starting client. %v\n", err)
		return
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM)
	<-exit

	cli.Finish()
}
