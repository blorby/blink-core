package main

import (
	"github.com/blinkops/blink-core/implementation"
	blinkSdk "github.com/blinkops/blink-sdk"
	"github.com/blinkops/blink-sdk/plugin/config"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

func main() {

	timestampFormat := "02-01-2006 15:04:05.00"
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: timestampFormat,
		FullTimestamp:   true,
	})

	// Get the current directory.
	currentDirectory, err := os.Getwd()
	if err != nil {
		log.Error("Failed getting current directory: ", err)
		panic(err)
	}

	log.Info("Current directory is: ", currentDirectory)

	// Initialize the configuration.
	err = os.Setenv(config.ConfigurationPathEnvVar, path.Join(currentDirectory, "config.yaml"))
	if err != nil {
		log.Error("Failed to set configuration env variable: ", err)
		panic(err)
	}

	plugin, err := implementation.NewCorePlugin(currentDirectory)
	if err != nil {
		log.Error("Failed to create plugin implementation: ", err)
		panic(err)
	}

	err = blinkSdk.Start(plugin)
	if err != nil {
		log.Fatal("Error during server startup: ", err)
	}
}
