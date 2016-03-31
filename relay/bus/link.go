package bus

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/operable/go-relay/relay"
)

const (
	COMMANDS_TOPIC_TEMPLATE = "bot/relays/%s/"
	EXECUTE_TOPIC_TEMPLATE  = "bot/relays/%s/exec"
)

type messageCB func(mqtt.Message)

type linkTopics struct {
	commands string
	execute  string
}

type Link struct {
	id        string
	config    *relay.CogInfo
	topics    *linkTopics
	conn      *mqtt.Client
	workQueue *relay.WorkQueue
	control   chan byte
	wg        sync.WaitGroup
}

func NewLink(id string, config *relay.CogInfo, workQueue *relay.WorkQueue, wg sync.WaitGroup) (*Link, error) {
	if id == "" || config == nil {
		err := errors.New("Relay id or Cog connection info is nil.")
		log.Fatal(err)
		return nil, err
	}
	link := &Link{id: id,
		config:    config,
		topics:    buildTopics(id),
		conn:      nil,
		workQueue: workQueue,
		control:   make(chan byte),
		wg:        wg,
	}
	if err := link.connect(); err != nil {
		return nil, err
	}
	return link, nil
}

func (link *Link) Run() error {
	if err := link.setupSubscriptions(); err != nil {
		log.Errorf("Error subscribing to required topics: %s", err)
		return err
	}
	link.wg.Add(1)
	defer link.wg.Done()
	<-link.control
	link.conn.Disconnect(15000)
	log.Info("Cog connection closed")
	return nil
}

func (link *Link) Halt() {
	link.control <- 1
}

func (link *Link) Call(data interface{}) (interface{}, error) {
	return nil, errors.New("Not implemented")
}

func buildTopics(id string) *linkTopics {
	return &linkTopics{
		commands: fmt.Sprintf(COMMANDS_TOPIC_TEMPLATE, id),
		execute:  fmt.Sprintf(EXECUTE_TOPIC_TEMPLATE, id),
	}
}

func (link *Link) setupSubscriptions() error {
	token := link.conn.Subscribe(link.topics.commands, 1, wrapCallback(link.handleCommand))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	token = link.conn.Subscribe(link.topics.execute, 1, wrapCallback(link.handleExecution))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (link *Link) handleExecution(message mqtt.Message) {
}

func (link *Link) handleCommand(message mqtt.Message) {
}

func (link *Link) connect() error {
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.SetKeepAlive(time.Duration(15) * time.Second)
	mqttOpts.SetPingTimeout(time.Duration(5) * time.Second)
	mqttOpts.SetConnectTimeout(time.Duration(5) * time.Second)
	mqttOpts.SetMaxReconnectInterval(time.Duration(10) * time.Second)
	mqttOpts.SetUsername(link.id)
	mqttOpts.SetPassword(link.config.Token)
	mqttOpts.SetClientID(link.id)
	url := fmt.Sprintf("tcp://%s:%d", link.config.Host, link.config.Port)
	mqttOpts.AddBroker(url)
	link.conn = mqtt.NewClient(mqttOpts)
	if token := link.conn.Connect(); token.Wait() && token.Error() != nil {
		link.conn = nil
		return token.Error()
	}
	return nil
}

func wrapCallback(callback messageCB) mqtt.MessageHandler {
	return func(client *mqtt.Client, message mqtt.Message) {
		callback(message)
	}
}
