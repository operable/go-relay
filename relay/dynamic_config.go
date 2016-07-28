package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/messages"
	"github.com/operable/go-relay/relay/util"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// DynamicConfigUpdater periodically updates bundle dynamic configurations from Cog
type DynamicConfigUpdater struct {
	id                string
	configTopic       string
	options           bus.ConnectionOptions
	conn              bus.Connection
	dynamicConfigRoot string
	lastSignature     string
	control           chan interface{}
	refreshInterval   time.Duration
	refreshTimer      *time.Timer
}

// NewDynamicConfigUpdater creates a new updater
func NewDynamicConfigUpdater(relayID string, busOpts bus.ConnectionOptions, dynamicConfigRoot string,
	refreshInterval time.Duration) *DynamicConfigUpdater {
	return &DynamicConfigUpdater{
		id:                relayID,
		configTopic:       fmt.Sprintf("bot/relays/%s/dynconfigs", relayID),
		options:           busOpts,
		dynamicConfigRoot: dynamicConfigRoot,
		refreshInterval:   refreshInterval,
		control:           make(chan interface{}),
	}
}

// Run connects the announcer to Cog and starts its main
// loop in a goroutine
func (dcu *DynamicConfigUpdater) Run() error {
	log.Infof("Managed bundle dynamic configs enabled.")
	log.Infof("Refreshing bundle dynamic configs every %v.", dcu.refreshInterval)
	dcu.options.AutoReconnect = true
	dcu.options.EventsHandler = dcu.handleBusEvents
	conn := &bus.MQTTConnection{}
	if err := conn.Connect(dcu.options); err != nil {
		return err
	}
	dcu.refreshConfigs()
	dcu.refreshTimer = time.AfterFunc(dcu.refreshInterval, dcu.refreshConfigs)
	go func() {
		dcu.wait()
	}()
	return nil
}

// Halt tells the DCU to stop.
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
	defer dcu.refreshTimer.Reset(dcu.refreshInterval)
	var envelope messages.DynamicConfigsResponseEnvelope
	decoder := util.NewJSONDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&envelope); err != nil {
		log.Errorf("Error decoding GetDynamicConfigs result: %s.", err)
	}
	if envelope.Signature != dcu.lastSignature && envelope.Changed == true {
		if dcu.updateConfigs(envelope.Signature, envelope.Configs) {
			dcu.lastSignature = envelope.Signature
			dcu.cleanOldConfigs()
			log.Info("Updated bundle dynamic configs.")

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
		dcu.refreshTimer.Reset(dcu.refreshInterval)
	}
}

func (dcu *DynamicConfigUpdater) wait() {
	<-dcu.control
	dcu.refreshTimer.Stop()
	dcu.conn.Disconnect()
}

// updateConfigs does its best to make dynamic config updates atomic. It does this by
// writing a new set of configs to a separate directory, creating a symlink pointing to
// the new config dir, and finally renaming the symlink to `dynamicConfigRoot`/config.ManagedDynamicConfigLink`.
// The process takes advantage of the atomic nature of renames on most sane OSs and
// filesystems.
func (dcu *DynamicConfigUpdater) updateConfigs(signature string, configs map[string][]messages.DynamicConfig) bool {
	if !dcu.verifyManagedConfigPath() {
		return false
	}
	updateDir := path.Join(dcu.dynamicConfigRoot, "..", signature)
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		log.Errorf("Error preparing directory %s for updated bundle dynamic configs: %s.", updateDir, err)
		return false
	}
	for bundle, bundleConfigLayers := range configs {
		if err := os.MkdirAll(path.Join(updateDir, bundle), 0755); err != nil {
			log.Errorf("Error preparing dynamic config for bundle %s: %s.", bundle, err)
			return false
		}

		for _, config := range bundleConfigLayers {
			convertedContents, err := yaml.Marshal(config.Config)
			if err != nil {
				log.Errorf("Error preparing dynamic config for bundle %s: %s.", bundle, err)
				return false
			}
			configFileName := path.Join(updateDir, bundle, configFileName(config))
			if err := ioutil.WriteFile(configFileName, convertedContents, 0644); err != nil {
				log.Errorf("Error writing dynamic config file to path %s: %s.", configFileName, err)
				return false
			}
			log.Debugf("Wrote bundle dynamic config file %s.", configFileName)
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

func (dcu *DynamicConfigUpdater) cleanOldConfigs() {
	entries, _ := filepath.Glob(path.Join(dcu.dynamicConfigRoot, "..", "*"))
	for _, entry := range entries {
		if entry == dcu.dynamicConfigRoot || strings.HasSuffix(entry, dcu.lastSignature) {
			continue
		}
		os.RemoveAll(entry)
	}
}

func (dcu *DynamicConfigUpdater) verifyManagedConfigPath() bool {
	info, err := os.Lstat(dcu.dynamicConfigRoot)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") {
			return true
		}
		log.Errorf("Error stat-ing dynamic config root directory: %s.", err)
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		log.Errorf("Managed dynamic config root directory %s is not a symlink. Update ABORTED.", dcu.dynamicConfigRoot)
		return false
	}
	return true
}

func configFileName(c messages.DynamicConfig) string {
	layer := strings.ToLower(c.Layer)
	if layer == "base" {
		return "config.yaml"
	}
	name := strings.ToLower(c.Name)
	return fmt.Sprintf("%s_%s.yaml", layer, name)
}
