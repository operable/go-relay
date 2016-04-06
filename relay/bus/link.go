package bus

import (
	"encoding/json"
	"errors"
	"fmt"
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

	//ExecuteTopicTemplate is a topic template used by Relays to
	// receive command execution requests from Cog
	ExecuteTopicTemplate = "bot/relays/%s/exec"

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

// Subscriptions describes the command and execution topics with their corresponding
// handler callbacks.
type Subscriptions struct {
	command          string
	execution        string
	CommandHandler   MessageCallback
	ExecutionHandler MessageCallback
}

// Link is a message bus connection. It's responsible for parsing incoming messages
// and queueing them onto relay.Queue.
type Link struct {
	id            string
	cogConfig     *config.CogInfo
	subscriptions Subscriptions
	conn          *mqtt.Client
	control       chan byte
	wg            sync.WaitGroup
}

// NewLink returns a new message bus link as a MessageBus reference.
func NewLink(id string, cogConfig *config.CogInfo, subscriptions Subscriptions, wg sync.WaitGroup) (MessageBus, error) {
	if id == "" || cogConfig == nil {
		err := errors.New("Relay id or Cog connection info is nil.")
		log.Fatal(err)
		return nil, err
	}
	link := &Link{id: id,
		cogConfig:     cogConfig,
		subscriptions: buildTopics(id, subscriptions),
		conn:          nil,
		control:       make(chan byte),
		wg:            wg,
	}
	return link, nil
}

// Run starts the Link instance in a separate goroutine.
func (link *Link) Run() error {
	if err := link.connect(); err != nil {
		return err
	}
	if err := link.setupSubscriptions(); err != nil {
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
	return link.subscriptions.command
}

func (link *Link) setupSubscriptions() error {
	token := link.conn.Subscribe(link.subscriptions.command, 1,
		wrapCallback(link, link.subscriptions.CommandHandler))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	token = link.conn.Subscribe(link.subscriptions.execution, 1,
		wrapCallback(link, link.subscriptions.ExecutionHandler))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (link *Link) connect() error {
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.SetKeepAlive(time.Duration(15) * time.Second)
	mqttOpts.SetPingTimeout(time.Duration(60) * time.Second)
	mqttOpts.SetConnectTimeout(time.Duration(5) * time.Second)
	mqttOpts.SetMaxReconnectInterval(time.Duration(10) * time.Second)
	mqttOpts.SetUsername(link.id)
	mqttOpts.SetPassword(link.cogConfig.Token)
	mqttOpts.SetClientID(link.id)
	mqttOpts.SetCleanSession(true)
	mqttOpts.SetWill(RelayDiscoveryTopic, newWill(link.id), 1, false)
	url := fmt.Sprintf("tcp://%s:%d", link.cogConfig.Host, link.cogConfig.Port)
	mqttOpts.AddBroker(url)
	link.conn = mqtt.NewClient(mqttOpts)
	if token := link.conn.Connect(); token.Wait() && token.Error() != nil {
		link.conn = nil
		return token.Error()
	}
	return nil
}

func wrapCallback(link *Link, callback MessageCallback) mqtt.MessageHandler {
	return func(client *mqtt.Client, message mqtt.Message) {
		callback(link, message.Topic(), message.Payload())
	}
}

func buildTopics(id string, subscriptions Subscriptions) Subscriptions {
	subscriptions.command = fmt.Sprintf(CommandTopicTemplate, id)
	subscriptions.execution = fmt.Sprintf(ExecuteTopicTemplate, id)
	return subscriptions
}

func newWill(id string) string {
	announcement := messages.NewAnnouncement(id, false)
	data, _ := json.Marshal(announcement)
	return string(data)
}
