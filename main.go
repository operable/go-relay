package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/docker"
)

var configFile = flag.String("file", "/etc/cog_relay.conf", "Path to configuration file")

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		DisableSorting:   true,
	})
}

func configureLogger(config *relay.Config) {
	if config.LogJSON == true {
		log.SetFormatter(&log.JSONFormatter{})
	}
	switch config.LogPath {
	case "stderr":
		log.SetOutput(os.Stderr)
	case "console":
		fallthrough
	case "stdout":
		log.SetOutput(os.Stdout)
	default:
		logFile, err := os.Open(config.LogPath)
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFile)
	}
	switch config.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "err":
		fallthrough
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		os.Stderr.Write([]byte(fmt.Sprintf("Unknown log level '%s'. Defaulting to info.\n",
			config.LogLevel)))
		log.SetLevel(log.InfoLevel)
	}
}

func prepare() *relay.Config {
	flag.Parse()
	config, err := relay.LoadConfig(*configFile)
	if err != nil {
		errstr := fmt.Sprintf("%s", err)
		msgs := strings.Split(errstr, ";")
		log.Errorf("Error loading %s:", *configFile)
		for _, msg := range msgs {
			log.Errorf("  %s", msg)
		}
		os.Exit(1)
	}
	configureLogger(config)
	return config
}

func main() {
	var coordinator sync.WaitGroup
	incomingSignal := make(chan os.Signal, 1)

	// Set up signal handlers
	signal.Notify(incomingSignal, syscall.SIGINT)
	config := prepare()
	log.Infof("Configuration file %s loaded.", *configFile)
	log.Infof("Relay %s is initializing.", config.ID)

	if config.DockerDisabled == false {
		err := docker.VerifyConfig(config.Docker)
		if err != nil {
			log.Errorf("Error verifying Docker configuration: %s.", err)
			os.Exit(3)
		} else {
			log.Infof("Docker configuration verified.")
		}
	} else {
		Logger.Infof("Docker support disabled.")
	}

	// Create work queue with some burstable capacity
	workQueue := relay.NewWorkQueue(config.MaxConcurrent * 2)

	// Start MaxConcurrent workers
	for i := 0; i < config.MaxConcurrent; i++ {
		go func() {
			relay.RunWorker(workQueue, coordinator)
		}()
	}

	log.Infof("Relay %s started %d workers.", config.ID, config.MaxConcurrent)
	log.Infof("Relay %s is ready.", config.ID)

	// Wait until we get a signal
	<-incomingSignal
	log.Infof("Relay %s is shutting down.", config.ID)

	// Remove signal handler so Ctrl-C works
	signal.Reset(syscall.SIGINT)

	// Stop work queue
	workQueue.Stop()

	// Wait on processes to exit
	coordinator.Wait()
	log.Infof("Relay %s shut down complete.", config.ID)
}
