package utils

import (
	"os"
	"os/signal"
	"syscall"
)

func MonitorSystemSignals(callback func(os.Signal)) {
	sigchnl := make(chan os.Signal, 1)

	signal.Notify(sigchnl, os.Interrupt, syscall.SIGTERM)

	for {
		s := <-sigchnl
		callback(s)
	}
}
