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
	err_result := client.ClientConfig{}
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
		return err_result, Err.Ctx("Could not parse CLI_LOOP-PERIOD_COMMENT env var as time.Duration.", err)
	}

	if _, err := time.ParseDuration(v.GetString("loop-period.post")); err != nil {
		return err_result, Err.Ctx("Could not parse CLI_LOOP-PERIOD_POST env var as time.Duration.", err)
	}

	c_config := client.ClientConfig{
		Id:                       v.GetUint("id"),
		Log_level:                v.GetString("log.level"),
		Server_address:           v.GetString("server.address"),
		Loop_period_post:         v.GetDuration("loop-period.post"),
		Loop_period_comment:      v.GetDuration("loop-period.comment"),
		File_path_post:           v.GetString("files-path.post"),
		File_path_comment:        v.GetString("files-path.comment"),
		File_path_sentiment_meme: v.GetString("files-path.sentiment-meme"),
		File_path_school_memes:   v.GetString("files-path.school-memes"),
	}

	return c_config, nil
}

func PrintConfig(c_config *client.ClientConfig) {
	log.Printf("Client %v configuration: \n", c_config.Id)
	log.Printf(" - ID: %v\n", c_config.Id)
	log.Printf(" - Server Address: %v\n", c_config.Server_address)
	log.Printf(" - Loop Period Post: %v\n", c_config.Loop_period_post)
	log.Printf(" - Loop Period Comment: %v\n", c_config.Loop_period_comment)
	log.Printf(" - File Path Post: %v\n", c_config.File_path_post)
	log.Printf(" - File Path Comment: %v\n", c_config.File_path_comment)
	log.Printf(" - File Path Sentiment Meme: %v\n", c_config.File_path_sentiment_meme)
	log.Printf(" - File Path School Memes: %v\n", c_config.File_path_school_memes)
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

	if err := InitLogger(config.Log_level); err != nil {
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
