package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
)

var configFile = flag.String("file", "/etc/cog_relay.conf", "Path to configuration file")
var cogHost = flag.String("cog_host", "", "Name of upstream Cog host")
var cogPort = flag.Int("cog_port", -1, "MQTT port of upstream Cog host")

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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

func main() {
	flag.Parse()
	config, err := relay.LoadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	configureLogger(config)
}
