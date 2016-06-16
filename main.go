package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/config"
)

const (
	BAD_CONFIG = iota + 1
	DOCKER_ERR
	BUS_ERR
)

var configFile = flag.String("file", "", "Path to configuration file")
var cpuprofile = flag.String("cpuprofile", "", "Write CPU profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

// Populated by build script
var buildstamp string
var buildhash string
var buildtag string

var configLocations = []string{
	"/etc/relay.conf",
	"/usr/local/etc/relay.conf",
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	displayVersionInfo()
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		DisableSorting:   true,
	})
}

func displayVersionInfo() {
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--version" || arg == "-version" {
			if buildtag == "" {
				buildtag = "<None>"
			}
			fmt.Printf("Git commit: %s\nGit tag: %s\nBuild timestamp: %s UTC\n",
				buildhash, buildtag, buildstamp)
			os.Exit(0)
		}
	}
}

func configureLogger(config *config.Config) {
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

func tryLoadingConfig(locations []string) config.RawConfig {
	for _, location := range locations {
		rawConfig, err := config.LoadConfig(location)
		if err != nil {
			log.Warnf("Error loading config file '%s': %s.", location, err)
			continue
		} else {
			log.Infof("Using config file '%s'.", location)
		}
		return rawConfig
	}
	return make(config.RawConfig, 0)
}

func prepare() *config.Config {
	flag.Parse()
	locations := configLocations
	if *configFile != "" {
		locations = []string{
			*configFile,
		}
	}

	rawConfig := tryLoadingConfig(locations)
	relayConfig, err := rawConfig.Parse()
	if err != nil {
		logMessage := ""
		msgs := []string{}
		for _, msg := range strings.Split(fmt.Sprintf("%s", err), ";") {
			if msg != "" {
				msgs = append(msgs, msg)
			}
		}
		count := len(msgs) - 1
		for i, msg := range msgs {
			if i < count {
				logMessage = fmt.Sprintf("%s  %s\n", logMessage, msg)
			} else {
				logMessage = fmt.Sprintf("%s  %s", logMessage, msg)
			}
		}
		log.Errorf("Error configuring Relay:\n%s", logMessage)
		log.Error("Relay start aborted.")
		os.Exit(BAD_CONFIG)
		return nil
	}
	configureLogger(relayConfig)
	return relayConfig
}

func main() {
	relayConfig := prepare()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			pprof.WriteHeapProfile(f)
			f.Close()
		}()
	}
	if err := relayConfig.Verify(); err != nil {
		log.Error(err)
		os.Exit(BAD_CONFIG)
		return
	}
	log.Infof("Relay %s is initializing.", relayConfig.ID)

	myRelay, err := relay.NewRelay(relayConfig)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	if err := myRelay.Start(); err != nil {
		log.Error(err)
		os.Exit(1)
		return
	}
	log.Infof("Relay %s online.", relayConfig.ID)
	// Set up signal handlers
	incomingSignal := make(chan os.Signal, 1)
	signal.Notify(incomingSignal, syscall.SIGINT)

	// Wait until we get a signal
	<-incomingSignal

	// Shutdown
	// Remove signal handler so Ctrl-C works
	signal.Reset(syscall.SIGINT)

	log.Info("Starting shut down.")
	myRelay.Stop()
	log.Infof("Relay %s shut down complete.", relayConfig.ID)
}
