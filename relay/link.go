package relay

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	BUFFERED_MSGS           = 2
	PRESENCE_TOPIC          = "bot/relays/presence"
	COMMANDS_TOPIC_TEMPLATE = "bot/relays/%s/commands"
	EXECUTE_TOPIC_TEMPLATE  = "bot/relays/%s/exec"
)

type messageCB func(mqtt.Message)

type LinkTopics struct {
	commands string
	execute  string
}

type Link struct {
	id      string
	config  *CogInfo
	topics  *LinkTopics
	conn    *mqtt.Client
	control chan int
	wg      *sync.WaitGroup
}

func NewLink(id string, config *CogInfo, wg *sync.WaitGroup) (*Link, error) {
	if id == "" || config == nil {
		err := errors.New("Relay id or Cog connection info is nil.")
		log.Fatal(err)
		return nil, err
	}
	return &Link{id: id,
		config:  config,
		topics:  buildTopics(id),
		conn:    nil,
		control: make(chan int, 1),
		wg:      wg,
	}, nil
}

func (link *Link) Run() error {
	if err := link.connect(); err != nil {
		return err
	}
	if err := link.setupSubscriptions(); err != nil {
		return err
	}
	link.wg.Add(1)
	go func() {
		defer link.wg.Done()
		<-link.control
		link.conn.Disconnect(15000)
		log.Info("Cog connection closed")
	}()
	return nil
}

func (link *Link) Stop() {
	link.control <- 1
}

func buildTopics(id string) *LinkTopics {
	return &LinkTopics{
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
	mqtt_opts := mqtt.NewClientOptions()
	mqtt_opts.SetKeepAlive(time.Duration(15) * time.Second)
	mqtt_opts.SetPingTimeout(time.Duration(5) * time.Second)
	mqtt_opts.SetUsername(link.id)
	mqtt_opts.SetPassword(link.config.Token)
	mqtt_opts.SetClientID(link.id)
	url := fmt.Sprintf("tcp://%s:%d", link.config.Host, link.config.Port)
	mqtt_opts.AddBroker(url)
	link.conn = mqtt.NewClient(mqtt_opts)
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
