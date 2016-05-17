package relay

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/operable/go-relay/relay/messages"
	"strconv"
	"sync"
	"time"
)

// Announcer announces Relay bundles lists to Cog
type Announcer interface {
	SendAnnouncement()
	Run() error
	Halt()
}

type relayAnnouncerCommand byte
type relayAnnouncerState byte

var errorStartAnnouncer = errors.New("Relay bundle announcer failed to start.")
var reannounceInterval = time.Duration(5) * time.Second

const (
	relayAnnouncerAnnounceCommand (relayAnnouncerCommand) = iota
	relayAnnouncerStopCommand
)

const (
	relayAnnouncerStoppedState (relayAnnouncerState) = iota
	relayAnnouncerWaitingState
	relayAnnouncerReceiptWaitingState
)

type relayAnnouncer struct {
	id                  string
	receiptTopic        string
	relay               *Relay
	options             *mqtt.ClientOptions
	conn                *mqtt.Client
	state               relayAnnouncerState
	stateLock           sync.Mutex
	control             chan relayAnnouncerCommand
	coordinator         sync.WaitGroup
	announcementID      uint64
	receiptFor          string
	announceTimer       *time.Timer
	announcementPending bool
}

// NewAnnouncer creates a new Announcer
func NewAnnouncer(relay *Relay, wg sync.WaitGroup) (Announcer, error) {
	busConfig := relay.Config.Cog
	announcer := &relayAnnouncer{
		id:                  relay.Config.ID,
		receiptTopic:        fmt.Sprintf("bot/relays/%s/announcer", relay.Config.ID),
		relay:               relay,
		options:             mqtt.NewClientOptions(),
		state:               relayAnnouncerStoppedState,
		control:             make(chan relayAnnouncerCommand, 2),
		coordinator:         wg,
		announcementPending: false,
	}
	announcer.options.SetAutoReconnect(false)
	announcer.options.SetKeepAlive(time.Duration(60) * time.Second)
	announcer.options.SetPingTimeout(time.Duration(15) * time.Second)
	announcer.options.SetCleanSession(true)
	announcer.options.SetClientID(fmt.Sprintf("%s-a", announcer.id))
	announcer.options.SetUsername(announcer.id)
	announcer.options.SetPassword(busConfig.Token)
	announcer.options.AddBroker(busConfig.URL())
	if busConfig.SSLEnabled == true {
		announcer.options.TLSConfig = tls.Config{
			ServerName:             busConfig.Host,
			SessionTicketsDisabled: true,
			InsecureSkipVerify:     false,
		}
	}

	return announcer, nil
}

// Run connects the announcer to Cog and starts its main
// loop in a goroutine
func (ra *relayAnnouncer) Run() error {
	ra.conn = mqtt.NewClient(ra.options)
	if token := ra.conn.Connect(); token.Wait() {
		err := token.Error()
		if err != nil {
			log.Errorf("1 Cog connection error: %s", err)
			return errorStartAnnouncer
		}
	}
	if token := ra.conn.Subscribe(ra.receiptTopic, 1, ra.cogReceipt); token.Wait() {
		err := token.Error()
		if err != nil {
			log.Errorf("2 Cog connection error: %s", err)
			ra.conn.Disconnect(0)
			return errorStartAnnouncer
		}
	}
	ra.state = relayAnnouncerWaitingState
	go func() {
		ra.loop()
	}()
	return nil
}

func (ra *relayAnnouncer) Halt() {
	ra.control <- relayAnnouncerStopCommand
}

func (ra *relayAnnouncer) SendAnnouncement() {
	ra.control <- relayAnnouncerAnnounceCommand
	log.Debug("Called relayAnnouncer.SendAnnouncement()")
}

func (ra *relayAnnouncer) disconnected(client *mqtt.Client, err error) {
	ra.Halt()
}

func (ra *relayAnnouncer) cogReceipt(client *mqtt.Client, message mqtt.Message) {
	receipt := messages.AnnouncementReceipt{}
	err := json.Unmarshal(message.Payload(), &receipt)
	if err != nil {
		log.Errorf("Ignoring illegal JSON receipt reply: %s.", err)
		return
	}
	ra.stateLock.Lock()
	defer ra.stateLock.Unlock()
	if receipt.ID != ra.receiptFor {
		log.Infof("Ignoring receipt for unknown bundle announcement %s.", receipt.ID)
	} else {
		epoch, _ := strconv.ParseUint(ra.receiptFor, 10, 64)
		ra.relay.bundles.EpochAcked(epoch)
		ra.receiptFor = ""
		if receipt.Status != "success" {
			log.Warnf("Cog returned unsuccessful status for bundle announcement %s: %s.", receipt.ID, receipt.Status)
		} else {
			log.Debugf("Cog successfully ack'd bundle announcement %s.", receipt.ID)
		}
		if ra.announcementPending == false {
			ra.state = relayAnnouncerWaitingState
			ra.announceTimer.Stop()
		}
	}
}

func (ra *relayAnnouncer) loop() {
	ra.coordinator.Add(1)
	defer ra.coordinator.Done()
	for ra.state != relayAnnouncerStoppedState {
		switch <-ra.control {
		case relayAnnouncerStopCommand:
			ra.state = relayAnnouncerStoppedState
			ra.announceTimer.Stop()
		case relayAnnouncerAnnounceCommand:
			ra.stateLock.Lock()
			if ra.state == relayAnnouncerReceiptWaitingState {
				ra.announcementPending = true
				ra.stateLock.Unlock()
			} else {
				ra.stateLock.Unlock()
				ra.sendAnnouncement(true)
				ra.announceTimer = time.AfterFunc(reannounceInterval, func() {
					log.Debugf("Retrying announcement %s.", ra.receiptFor)
					ra.sendAnnouncement(false)
				})
			}
		}
	}
}

func (ra *relayAnnouncer) sendAnnouncement(skipTimer bool) {
	ra.stateLock.Lock()
	defer ra.stateLock.Unlock()
	log.Debug("Preparing announcement")
	// We're waiting on Cog to reply to a previous announcement
	announcementID := fmt.Sprintf("%d", ra.relay.bundles.CurrentEpoch())
	announcement := messages.NewBundleAnnouncementExtended(ra.id, ra.relay.GetBundles(), ra.receiptTopic, announcementID)
	raw, _ := json.Marshal(announcement)
	for {
		log.Debug("Publishing bundle announcement to bot/relays/discover")
		if token := ra.conn.Publish("bot/relays/discover", 1, false, raw); token.Wait() && token.Error() != nil {
			ra.stateLock.Unlock()
			log.Debug("Retrying announcement")
			time.Sleep(time.Duration(1) * time.Second)
			ra.stateLock.Lock()
		} else {
			log.Debugf("Announcement sent.")
			break
		}
	}
	ra.receiptFor = announcementID
	ra.state = relayAnnouncerReceiptWaitingState
	if skipTimer == false {
		ra.announceTimer.Reset(reannounceInterval)
	}
}
