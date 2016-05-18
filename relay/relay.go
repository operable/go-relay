package relay

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/bundle"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"github.com/operable/go-relay/relay/util"
	"github.com/operable/go-relay/relay/worker"
	"golang.org/x/net/context"
	"strings"
)

const (
	// infoTopic is a topic used by Relays to ask Cog for information
	// such as the list of assigned bundles
	infoTopic = "bot/relays/info"

	// directiveTopicTemplate is a topic template used by Relays to
	// receive Relay directives from Cog
	directiveTopicTemplate = "bot/relays/%s/directives"

	// commandTopicTemplate is a topic template used by Relays to
	// receive command execution requests from Cog
	commandTopicTemplate = "/bot/commands/%s/#"
)

// Relay is responsible for connecting to the message bus
// and processing directly or dispatching to a worker pool
// any incoming messages.
type Relay interface {
	Start() error
	Stop() error
}

type cogRelay struct {
	config            *config.Config
	connOpts          bus.ConnectionOptions
	conn              bus.Connection
	queue             util.Queue
	catalog           *bundle.Catalog
	announcer         Announcer
	directivesReplyTo string
}

// NewRelay constructs a new Relay instance
func NewRelay(config *config.Config) (Relay, error) {
	if err := verifyDockerConfig(config); err != nil {
		return nil, err
	}
	return &cogRelay{
		config:            config,
		catalog:           bundle.NewCatalog(),
		queue:             util.NewQueue(uint(config.MaxConcurrent)),
		directivesReplyTo: fmt.Sprintf(directiveTopicTemplate, config.ID),
	}, nil
}

func (r *cogRelay) Start() error {
	r.connOpts = bus.ConnectionOptions{
		Userid:      r.config.ID,
		Password:    r.config.Cog.Token,
		Host:        r.config.Cog.Host,
		Port:        r.config.Cog.Port,
		SSLEnabled:  r.config.Cog.SSLEnabled,
		SSLCertPath: r.config.Cog.SSLCertPath,
		OnDisconnect: &bus.DisconnectMessage{
			Topic: "bot/relays/discover",
			Body:  newWill(r.config.ID),
		},
		EventsHandler: r.handleBusEvents,
	}
	for i := 0; i < r.config.MaxConcurrent; i++ {
		go func() {
			worker.ExecutionWorker(r.queue)
		}()
	}
	log.Infof("Started %d request workers.", r.config.MaxConcurrent)
	conn := &bus.MQTTConnection{}
	if err := conn.Connect(r.connOpts); err != nil {
		return err
	}
	return nil
}

func (r *cogRelay) Stop() error {
	return nil
}

func (r *cogRelay) handleBusEvents(conn bus.Connection, event bus.Event) {
	if event == bus.ConnectedEvent {
		r.conn = conn
		if r.announcer == nil {
			opts := r.connOpts
			opts.EventsHandler = nil
			opts.OnDisconnect = nil
			r.announcer = NewAnnouncer(r.config.ID, opts, r.catalog)
			if err := r.announcer.Run(); err != nil {
				log.Errorf("Failed to start announcer: %s.", err)
				panic(err)
			}
		}
		if err := r.setSubscriptions(); err != nil {
			log.Errorf("Failed to set Relay subscriptions: %s.", err)
			panic(err)
		}
		if r.catalog.Len() > 0 {
			r.catalog.Reconnected()
		}
		log.Info("Loading bundle catalog.")
		r.requestBundles()
	}
}

func (r *cogRelay) setSubscriptions() error {
	// Set directives handler
	if err := r.conn.Subscribe(fmt.Sprintf(directiveTopicTemplate, r.config.ID), r.handleDirective); err != nil {
		return err
	}
	return r.conn.Subscribe(fmt.Sprintf(commandTopicTemplate, r.config.ID), r.handleCommand)
}

func (r *cogRelay) handleCommand(conn bus.Connection, topic string, message []byte) {
	log.Debugf("Got invocation request on %s", topic)
	invoke := &worker.CommandInvocation{
		RelayConfig: *r.config,
		Publisher:   r.conn,
		Catalog:     *r.catalog,
		Topic:       topic,
		Payload:     message,
	}
	ctx := context.WithValue(context.Background(), "invoke", invoke)
	log.Debugf("Queue stopped: %v", r.queue.IsStopped())
	log.Debugf("Eqneueud request: %s", r.queue.Enqueue(ctx))
	log.Debugf("Enqueued invocation request for %s", topic)
}

func (r *cogRelay) handleDirective(conn bus.Connection, topic string, message []byte) {
	tm, err := messages.ParseUntypedDirective(message)
	if err != nil {
		log.Errorf("Ignoring bad directive message: %s", err)
		return
	}
	// Dispatch on mesasge type
	switch tm.(type) {
	case *messages.ListBundlesResponseEnvelope:
		log.Info("Processing bundle list")
		r.updateCatalog(tm.(*messages.ListBundlesResponseEnvelope))
	}
}

func (r *cogRelay) updateCatalog(envelope *messages.ListBundlesResponseEnvelope) {
	bundles := []*config.Bundle{}
	for _, b := range envelope.Bundles {
		if b.Enabled {
			b.ConfigFile.Version = fixBundleVersion(b.ConfigFile.Version)
			bundles = append(bundles, &b.ConfigFile)
		}
	}
	if !r.queue.IsStopped() {
		events := make(util.QueueEvents)
		if r.queue.Stop(events) == true {
			<-events
		} else {
			log.Error("Failed to stop worker queue. Bundle catalog update aborted.")
			return
		}
	}
	defer r.queue.Start()
	// TODO: This should be bi-directional sync to catch bundle unassignments too
	r.catalog.AddBatch(bundles)
	if r.catalog.IsChanged() {
		r.refreshBundles()
		log.Info("Changes to bundle assignments detected.")
		r.announcer.SendAnnouncement()
	}
}

func (r *cogRelay) refreshBundles() error {
	dockerEngine, err := engines.NewDockerEngine(*r.config)
	if err != nil {
		return err
	}
	for _, name := range r.catalog.BundleNames() {
		if bundle := r.catalog.FindLatest(name); bundle != nil {
			if bundle.NeedsRefresh() {
				if bundle.IsDocker() {
					avail, _ := dockerEngine.IsAvailable(bundle.Docker.Image, bundle.Docker.Tag)
					bundle.SetAvailable(avail)
				} else {
					r.catalog.MarkReady(bundle.Name, bundle.Version)
				}
			}
		}
	}
	return nil
}

func (r *cogRelay) requestBundles() error {
	msg := messages.ListBundlesEnvelope{
		ListBundles: &messages.ListBundlesMessage{
			RelayID: r.config.ID,
			ReplyTo: r.directivesReplyTo,
		},
	}
	raw, _ := json.Marshal(&msg)
	log.Info("Refreshing command bundles.")
	return r.conn.Publish(infoTopic, raw)
}

func verifyDockerConfig(c *config.Config) error {
	if c.DockerEnabled() == true {
		if err := engines.VerifyDockerConfig(c.Docker); err != nil {
			log.Errorf("Error verifying Docker configuration: %s.", err)
			return err
		}
		log.Infof("Docker configuration verified.")
	} else {
		log.Infof("Docker support disabled.")
	}
	return nil
}

func fixBundleVersion(version string) string {
	if len(strings.Split(version, ".")) == 2 {
		return fmt.Sprintf("%s.0", version)
	}
	return version
}

func newWill(id string) string {
	announcement := messages.NewAnnouncement(id, false)
	data, _ := json.Marshal(announcement)
	return string(data)
}
