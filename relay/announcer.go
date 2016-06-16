package relay

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/bundle"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
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
var reannounceInterval2 = time.Duration(5) * time.Second

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
	options             bus.ConnectionOptions
	conn                bus.Connection
	catalog             *bundle.Catalog
	state               relayAnnouncerState
	stateLock           sync.Mutex
	control             chan relayAnnouncerCommand
	receiptFor          string
	announceTimer       *time.Timer
	announcementPending bool
}

// NewAnnouncer creates a new Announcer
func NewAnnouncer(relayID string, busOpts bus.ConnectionOptions, catalog *bundle.Catalog) Announcer {
	announcer := &relayAnnouncer{
		id:                  relayID,
		receiptTopic:        fmt.Sprintf("bot/relays/%s/announcer", relayID),
		options:             busOpts,
		catalog:             catalog,
		state:               relayAnnouncerStoppedState,
		control:             make(chan relayAnnouncerCommand, 2),
		announcementPending: false,
	}
	return announcer
}

// Run connects the announcer to Cog and starts its main
// loop in a goroutine
func (ra *relayAnnouncer) Run() error {
	ra.options.OnDisconnect = &bus.DisconnectMessage{
		Topic: "bot/relays/discover",
		Body:  newWill(ra.id, ra.receiptTopic),
	}
	ra.options.EventsHandler = ra.handleBusEvents
	ra.options.OnDisconnect = &bus.DisconnectMessage{
		Topic: "bot/relays/discover",
		Body:  newWill(ra.id, ra.receiptTopic),
	}
	conn := &bus.MQTTConnection{}
	if err := conn.Connect(ra.options); err != nil {
		return err
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

func (ra *relayAnnouncer) handleBusEvents(conn bus.Connection, event bus.Event) {
	if event == bus.ConnectedEvent {
		ra.conn = conn
		if err := ra.conn.Subscribe(ra.receiptTopic, ra.cogReceipt); err != nil {
			log.Errorf("Failed to set up announcer subscriptions: %s.", err)
			panic(err)
		}
	}

}

func (ra *relayAnnouncer) cogReceipt(conn bus.Connection, topic string, payload []byte) {
	receipt := messages.AnnouncementReceipt{}
	err := json.Unmarshal(payload, &receipt)
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
		ra.catalog.EpochAcked(epoch)
		ra.receiptFor = ""
		if receipt.Status != "success" {
			log.Warnf("Cog returned unsuccessful status for bundle announcement %s: %s.", receipt.ID, receipt.Status)
		} else {
			log.Infof("Cog successfully ack'd bundle announcement %s.", receipt.ID)
		}
		if ra.announcementPending == false {
			ra.state = relayAnnouncerWaitingState
			ra.announceTimer.Stop()
		}
	}
}

func (ra *relayAnnouncer) loop() {
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
				ra.announceTimer = time.AfterFunc(reannounceInterval2, func() {
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
	announcementID := fmt.Sprintf("%d", ra.catalog.CurrentEpoch())
	announcement := messages.NewBundleAnnouncementExtended(ra.id, getBundles(ra.catalog), ra.receiptTopic, announcementID)
	raw, _ := json.Marshal(announcement)
	for {
		log.Debug("Publishing bundle announcement to bot/relays/discover")
		if err := ra.conn.Publish("bot/relays/discover", raw); err != nil {
			ra.stateLock.Unlock()
			log.Error(err)
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
		ra.announceTimer.Reset(reannounceInterval2)
	}
}

func getBundles(catalog *bundle.Catalog) []config.Bundle {
	names := catalog.BundleNames()
	var retval []config.Bundle
	for _, name := range names {
		bundle := catalog.Find(name)
		if bundle != nil && bundle.IsAvailable() {
			retval = append(retval, *bundle)
		}
	}
	return retval
}

func newWill(id string, replyTo string) string {
	announcement := messages.NewOfflineAnnouncement(id, replyTo)
	data, _ := json.Marshal(announcement)
	return string(data)
}
