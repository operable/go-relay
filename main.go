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
	log.Infof("%s loaded.", *configFile)
	log.Infof("Relay %s initializing", config.ID)

	docker, err := relay.NewDockerEngine(config.Docker, &coordinator)
	if err != nil {
		log.Errorf("Error initializing Docker execution engine: %s", err)
		os.Exit(2)
	}

	err = docker.Run()
	if err != nil {
		os.Exit(2)
	}

	link, err := relay.NewLink(config.ID, config.Cog, &coordinator)
	if err != nil {
		os.Exit(4)
	}
	err = link.Run()
	if err != nil {
		os.Exit(4)
	}
	log.Infof("Relay %s ready", config.ID)
	// Wait until we receive a signal
	<-incomingSignal

	// Remove signal handler so ctrl-C works
	signal.Reset(syscall.SIGINT)

	log.Infof("Relay %s signing off", config.ID)
	link.Halt()
	docker.Halt()
	coordinator.Wait()
	log.Infof("Relay %s shut down", config.ID)
}
