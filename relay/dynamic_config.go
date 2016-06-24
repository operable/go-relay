package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/messages"
	"io/ioutil"
	"os"
	"path"
	"time"
)

var refreshInterval = time.Duration(5) * time.Second

// DynamicConfigUpdater periodically updates bundle dynamic configurations from Cog
type DynamicConfigUpdater struct {
	id                string
	configTopic       string
	options           bus.ConnectionOptions
	conn              bus.Connection
	dynamicConfigRoot string
	lastSignature     string
	control           chan interface{}
	refreshTimer      *time.Timer
}

// NewDynamicConfigUpdater creates a new updater
func NewDynamicConfigUpdater(relayID string, busOpts bus.ConnectionOptions, dynamicConfigRoot string) *DynamicConfigUpdater {
	return &DynamicConfigUpdater{
		id:                relayID,
		configTopic:       fmt.Sprintf("bot/relays/%s/dynconfigs", relayID),
		options:           busOpts,
		dynamicConfigRoot: dynamicConfigRoot,
		control:           make(chan interface{}),
	}
}

// Run connects the announcer to Cog and starts its main
// loop in a goroutine
func (dcu *DynamicConfigUpdater) Run() error {
	dcu.options.AutoReconnect = true
	dcu.options.EventsHandler = dcu.handleBusEvents
	conn := &bus.MQTTConnection{}
	if err := conn.Connect(dcu.options); err != nil {
		return err
	}
	dcu.refreshTimer = time.AfterFunc(refreshInterval, dcu.refreshConfigs)
	go func() {
		dcu.loop()
	}()
	return nil
}

func (dcu *DynamicConfigUpdater) Halt() {
	dcu.control <- 1
}

func (dcu *DynamicConfigUpdater) handleBusEvents(conn bus.Connection, event bus.Event) {
	if event == bus.ConnectedEvent {
		dcu.conn = conn
		if err := dcu.conn.Subscribe(dcu.configTopic, dcu.dynConfigUpdate); err != nil {
			log.Errorf("Failed to set up dynamic config updater subscriptions: %s.", err)
			panic(err)
		}
	}
}

func (dcu *DynamicConfigUpdater) dynConfigUpdate(conn bus.Connection, topic string, payload []byte) {
	defer dcu.refreshTimer.Reset(refreshInterval)
	var envelope messages.DynamicConfigsResponseEnvelope
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		log.Errorf("Error decoding GetDynamicConfigs result: %s.", err)
	}
	if envelope.Signature != dcu.lastSignature && envelope.Changed == true {
		if dcu.updateConfigs(envelope.Signature, envelope.Configs) {
			if dcu.lastSignature != "" {
				if err := os.RemoveAll(dcu.lastSignature); err != nil {
					log.Warnf("Fafiled to clean up old bundle dynamic configs: %s.", err)
				}
			}
			dcu.lastSignature = envelope.Signature
		}
	}
}

func (dcu *DynamicConfigUpdater) refreshConfigs() {
	request := messages.GetDynamicConfigsEnvelope{
		GetDynamicConfigs: &messages.GetDynamicConfigs{
			RelayID:   dcu.id,
			ReplyTo:   dcu.configTopic,
			Signature: dcu.lastSignature,
		},
	}
	raw, _ := json.Marshal(request)
	if err := dcu.conn.Publish("bot/relays/info", raw); err != nil {
		log.Errorf("Error requesting bundle dynamic configuration update: %s.", err)
		dcu.refreshTimer.Reset(refreshInterval)
	}
}

func (dcu *DynamicConfigUpdater) loop() {
	<-dcu.control
	dcu.refreshTimer.Stop()
	dcu.conn.Disconnect()
}

func (dcu *DynamicConfigUpdater) updateConfigs(signature string, configs []messages.DynamicConfig) bool {
	updateDir := path.Join(dcu.dynamicConfigRoot, "..", signature)
	os.RemoveAll(updateDir)
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		log.Errorf("Error preparing directory %s for updated bundle dynamic configs: %s.", updateDir, err)
		return false
	}
	for _, config := range configs {
		convertedContents, err := yaml.Marshal(config.Config)
		if err != nil {
			log.Errorf("Error preparing dynamic config for bundle %s: %s.", config.BundleName, err)
			return false
		}
		if err := os.MkdirAll(path.Join(updateDir, config.BundleName), 0755); err != nil {
			log.Errorf("Error preparing dynamic config for bundle %s: %s.", config.BundleName, err)
			return false
		}
		configFileName := path.Join(updateDir, config.BundleName, "config.yml")
		if err := ioutil.WriteFile(configFileName, convertedContents, 0644); err != nil {
			log.Errorf("Error writing dynamic config file to path %s: %s.", configFileName, err)
			return false
		}
	}
	// Create and rename new symlink should make config updates atomic
	symlinkTarget := path.Join(dcu.dynamicConfigRoot, "..", "new")
	if err := os.Symlink(updateDir, symlinkTarget); err != nil {
		log.Errorf("Error replacing existing bundle dynamic configs with updated contents: %s.", err)
		return false
	}
	if err := os.Rename(symlinkTarget, dcu.dynamicConfigRoot); err != nil {
		log.Errorf("Error replacing existing bundle dynamic configs with updated contents: %s.", err)
		return false
	}
	return true
}
