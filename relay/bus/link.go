package bus

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
)

const (
	// CommandTopicTemplate is a topic template used by Relays to
	// receive Relay directives from Cog
	CommandTopicTemplate = "bot/relays/%s/directives"

	// ExecuteTopicTemplate is a topic template used by Relays to
	// receive command execution requests from Cog
	ExecuteTopicTemplate = "/bot/commands/%s/#"

	// RelayDiscoveryTopic is the topic Cog uses to discover new Relays
	RelayDiscoveryTopic = "bot/relays/discover"
)

// MessageBus is the interface used by worker code to
// publish messages
type MessageBus interface {
	Run() error
	Halt()
	Publish(topic string, payload []byte) error
	DirectiveReplyTo() string
}

// MessageCallback describes the function signature used to process messages.
// It is used for Relay directives and command execution.
type MessageCallback func(bus MessageBus, topic string, payload []byte)

// DisconnectCallback is called when the MQTT connection drops unexpectedly
type DisconnectCallback func(error)

// Handlers describes the command and execution topics with their corresponding
// handler callbacks.
type Handlers struct {
	command           string
	execution         string
	CommandHandler    MessageCallback
	ExecutionHandler  MessageCallback
	DisconnectHandler DisconnectCallback
}

// Link is a message bus connection.
type Link struct {
	id        string
	cogConfig *config.CogInfo
	handlers  Handlers
	conn      *mqtt.Client
	control   chan byte
	wg        sync.WaitGroup
}

// NewLink returns a new message bus link as a MessageBus reference.
func NewLink(id string, cogConfig *config.CogInfo, handlers Handlers, wg sync.WaitGroup) (MessageBus, error) {
	if id == "" || cogConfig == nil {
		err := errors.New("Relay id or Cog connection info is nil.")
		log.Fatal(err)
		return nil, err
	}
	link := &Link{id: id,
		cogConfig: cogConfig,
		handlers:  buildTopics(id, handlers),
		conn:      nil,
		control:   make(chan byte),
		wg:        wg,
	}
	return link, nil
}

// Run starts the Link instance in a separate goroutine.
func (link *Link) Run() error {
	if err := link.connect(); err != nil {
		return err
	}
	if err := link.setupHandlers(); err != nil {
		log.Errorf("Error subscribing to required topics: %s.", err)
		return err
	}
	go func() {
		link.wg.Add(1)
		defer link.wg.Done()
		<-link.control
		link.conn.Disconnect(15000)
		log.Info("Cog connection closed.")
	}()
	return nil
}

// Halt stops a running Link instance.
func (link *Link) Halt() {
	link.control <- 1
}

// Publish publishes a message onto the underlying message bus.
func (link *Link) Publish(topic string, payload []byte) error {
	if token := link.conn.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// DirectiveReplyTo is a message bus topic used by Relay to
// receive Cog responses
func (link *Link) DirectiveReplyTo() string {
	return link.handlers.command
}

func (link *Link) setupHandlers() error {
	token := link.conn.Subscribe(link.handlers.command, 1,
		wrapCallback(link, link.handlers.CommandHandler))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	token = link.conn.Subscribe(link.handlers.execution, 1,
		wrapCallback(link, link.handlers.ExecutionHandler))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (link *Link) connect() error {
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.SetAutoReconnect(false)
	if link.handlers.DisconnectHandler != nil {
		handler := func(client *mqtt.Client, err error) {
			link.handlers.DisconnectHandler(err)
		}
		mqttOpts.SetConnectionLostHandler(handler)
	}
	if link.cogConfig.SSLEnabled == true {
		mqttOpts.TLSConfig = tls.Config{
			ServerName:             link.cogConfig.Host,
			SessionTicketsDisabled: true,
			InsecureSkipVerify:     false,
		}
	}
	mqttOpts.SetKeepAlive(time.Duration(60) * time.Second)
	mqttOpts.SetPingTimeout(time.Duration(15) * time.Second)
	mqttOpts.SetUsername(link.id)
	mqttOpts.SetPassword(link.cogConfig.Token)
	mqttOpts.SetClientID(link.id)
	mqttOpts.SetCleanSession(true)
	mqttOpts.SetWill(RelayDiscoveryTopic, newWill(link.id), 1, false)
	url := fmt.Sprintf("tcp://%s:%d", link.cogConfig.Host, link.cogConfig.Port)
	mqttOpts.AddBroker(url)
	link.conn = mqtt.NewClient(mqttOpts)
	passes, wait := nextIncrement(0)
	for {
		if token := link.conn.Connect(); token.Wait() && token.Error() != nil {
			log.Error("Cog connection error.")
			log.Infof("Waiting %d seconds before retrying.", wait)
			time.Sleep(time.Duration(wait) * time.Second)
			passes, wait = nextIncrement(passes)
		} else {
			break
		}
	}
	return nil
}

func wrapCallback(link *Link, callback MessageCallback) mqtt.MessageHandler {
	return func(client *mqtt.Client, message mqtt.Message) {
		callback(link, message.Topic(), message.Payload())
	}
}

func buildTopics(id string, handlers Handlers) Handlers {
	handlers.command = fmt.Sprintf(CommandTopicTemplate, id)
	handlers.execution = fmt.Sprintf(ExecuteTopicTemplate, id)
	return handlers
}

func newWill(id string) string {
	announcement := messages.NewAnnouncement(id, false)
	data, _ := json.Marshal(announcement)
	return string(data)
}

func prepareForRetries() {
	// Seed rand to reduce determinism
	now := time.Now()
	rand.Seed(now.UnixNano())
}

func nextIncrement(i int) (int, int) {
	switch i {
	case 0:
		prepareForRetries()
		fallthrough
	case 1:
		return i + 1, 5 + rand.Intn(11)
	case 2:
		fallthrough
	case 3:
		return i + 1, 10 + rand.Intn(23)
	case 4:
		fallthrough
	case 5:
		return i + 1, 30 + rand.Intn(47)
	case 6:
		fallthrough
	case 7:
		return i + 1, 60 + rand.Intn(97)
	default:
		return i, 90 + rand.Intn(11)
	}
}
