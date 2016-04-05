package main

import (
	"encoding/json"
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
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"github.com/operable/go-relay/relay/worker"
	"golang.org/x/net/context"
)

const (
	BAD_CONFIG = iota + 1
	DOCKER_ERR
	BUS_ERR
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

func prepare() *config.Config {
	flag.Parse()
	relayConfig, err := config.LoadConfig(*configFile)
	if err != nil {
		errstr := fmt.Sprintf("%s", err)
		msgs := strings.Split(errstr, ";")
		log.Errorf("Error loading %s:", *configFile)
		for _, msg := range msgs {
			log.Errorf("  %s", msg)
		}
		os.Exit(BAD_CONFIG)
	}
	configureLogger(relayConfig)
	return relayConfig
}

func shutdown(relayConfig *config.Config, link relay.MessageBus, workQueue *relay.Queue, coordinator sync.WaitGroup) {
	// Remove signal handler so Ctrl-C works
	signal.Reset(syscall.SIGINT)

	log.Info("Starting shut down.")

	// Stop message bus listeners
	if link != nil {
		link.Halt()
	}

	// Stop work queue
	workQueue.Stop()

	// Wait on processes to exit
	coordinator.Wait()
	log.Infof("Relay %s shut down complete.", relayConfig.ID)
}

func askForInstalledBundles(relayConfig *config.Config, msgbus relay.MessageBus) error {
	msg := messages.ListBundlesEnvelope{
		ListBundles: &messages.ListBundlesMessage{
			RelayID: relayConfig.ID,
			ReplyTo: msgbus.DirectiveReplyTo(),
		},
	}
	raw, _ := json.Marshal(&msg)
	log.Info("Requested latest command bundle assignments.")
	return msgbus.Publish("bot/relays/info", raw)
}

func main() {
	var coordinator sync.WaitGroup
	incomingSignal := make(chan os.Signal, 1)

	// Set up signal handlers
	signal.Notify(incomingSignal, syscall.SIGINT)
	relayConfig := prepare()
	log.Infof("Configuration file %s loaded.", *configFile)
	log.Infof("Relay %s is initializing.", relayConfig.ID)

	// Create work queue with some burstable capacity
	workQueue := relay.NewQueue(relayConfig.MaxConcurrent * 2)

	if relayConfig.DockerDisabled == false {
		err := engines.VerifyDockerConfig(relayConfig.Docker)
		if err != nil {
			log.Errorf("Error verifying Docker configuration: %s.", err)
			shutdown(relayConfig, nil, workQueue, coordinator)
			os.Exit(DOCKER_ERR)
		} else {
			log.Infof("Docker configuration verified.")
		}
	} else {
		log.Infof("Docker support disabled.")
	}
	// Start MaxConcurrent workers
	for i := 0; i < relayConfig.MaxConcurrent; i++ {
		go func() {
			worker.RunWorker(workQueue, coordinator)
		}()
	}
	log.Infof("Started %d workers.", relayConfig.MaxConcurrent)

	// Handler func used for both message types
	handler := func(bus relay.MessageBus, topic string, payload []byte) {
		handleMessage(workQueue, relayConfig, bus, topic, payload)
	}

	// Connect to Cog
	subs := bus.Subscriptions{
		CommandHandler:   handler,
		ExecutionHandler: handler,
	}
	link, err := bus.NewLink(relayConfig.ID, relayConfig.Cog, workQueue, subs, coordinator)
	if err != nil {
		log.Errorf("Error connecting to Cog: %s.", err)
		shutdown(relayConfig, nil, workQueue, coordinator)
		os.Exit(BUS_ERR)
	}

	log.Infof("Connected to Cog host %s.", relayConfig.Cog.Host)
	err = link.Run()
	if err != nil {
		log.Errorf("Error subscribing to message topics: %s.", err)
		shutdown(relayConfig, nil, workQueue, coordinator)
		os.Exit(BUS_ERR)
	}
	err = askForInstalledBundles(relayConfig, link)
	if err != nil {
		log.Errorf("Error initiating installed bundles handshake: %s.", err)
		os.Exit(BUS_ERR)
	}
	log.Infof("Relay %s is ready.", relayConfig.ID)

	// Wait until we get a signal
	<-incomingSignal

	// Shutdown
	shutdown(relayConfig, link, workQueue, coordinator)
}

func handleMessage(queue *relay.Queue, relayConfig *config.Config, bus relay.MessageBus, topic string, payload []byte) {
	message := &relay.Incoming{
		Config:  relayConfig,
		Bus:     bus,
		Topic:   topic,
		Payload: payload,
	}
	ctx := context.WithValue(context.Background(), "message", message)
	queue.Enqueue(ctx)
}
