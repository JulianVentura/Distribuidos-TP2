package utils

import (
	"os"
	"os/signal"
	"syscall"
)

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
