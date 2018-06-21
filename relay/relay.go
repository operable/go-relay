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
	"github.com/operable/go-relay/relay/worker"
	"golang.org/x/net/context"
	"strings"
	"time"
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
	queue             chan interface{}
	engines           *engines.Engines
	dockerEngine      engines.Engine
	catalog           *bundle.Catalog
	announcer         Announcer
	dynConfigUpdater  *DynamicConfigUpdater
	directivesReplyTo string
	bundleTimer       *time.Timer
	cleanTimer        *time.Timer
}

// NewRelay constructs a new Relay instance
func NewRelay(config *config.Config) (Relay, error) {
	return &cogRelay{
		config:            config,
		engines:           engines.NewEngines(config),
		catalog:           bundle.NewCatalog(),
		queue:             make(chan interface{}, config.MaxConcurrent),
		directivesReplyTo: fmt.Sprintf(directiveTopicTemplate, config.ID),
	}, nil
}

func (r *cogRelay) Start() error {
	if r.config.DockerEnabled() == true {
		dockerEngine, err := r.engines.GetEngine(engines.DockerEngineType)
		if err != nil {
			return err
		}
		if err := dockerEngine.Init(); err != nil {
			return err
		}
		r.dockerEngine = dockerEngine
	}
	r.connOpts = r.makeConnOpts()
	r.connOpts.Userid = fmt.Sprintf("%s/announcer", r.config.ID)
	r.connOpts.EventsHandler = r.handleBusEvents
	r.connOpts.OnDisconnect = &bus.DisconnectMessage{
		Topic: "bot/relays/discover",
		Body:  newWill(r.config.ID, fmt.Sprintf("bot/relays/%s/announcer", r.config.ID)),
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
	if r.config.DockerEnabled() {
		r.cleanTimer = time.AfterFunc(r.config.Docker.CleanDuration(), r.scheduledDockerCleanup)
		log.Infof("Cleaning up expired Docker environments every %v.", r.config.Docker.CleanDuration())
	}
	log.Infof("Refreshing bundle catalog every %v.", r.config.RefreshDuration())
	return nil
}

func (r *cogRelay) Stop() error {
	if r.bundleTimer != nil {
		r.bundleTimer.Stop()
	}
	if r.config.DockerEnabled() {
		if r.bundleTimer != nil {
			r.cleanTimer.Stop()
		}
	}
	if r.announcer != nil {
		r.announcer.Halt()
	}
	if r.dynConfigUpdater != nil {
		r.dynConfigUpdater.Halt()
	}
	return nil
}

func (r *cogRelay) handleBusEvents(conn bus.Connection, event bus.Event) {
	if event == bus.ConnectedEvent {
		r.conn = conn
		if r.announcer == nil {
			r.announcer = NewAnnouncer(r.config.ID, r.conn, r.catalog)
			if err := r.announcer.Run(); err != nil {
				log.Errorf("Failed to start announcer: %s.", err)
				panic(err)
			}
			if r.config.ManagedDynamicConfig == true {
				opts := r.makeConnOpts()
				r.dynConfigUpdater = NewDynamicConfigUpdater(r.config.ID, opts, r.config.DynamicConfigRoot,
					r.config.ManagedDynamicConfigRefreshDuration())
				if err := r.dynConfigUpdater.Run(); err != nil {
					log.Errorf("Failed to start bundle dynamic config updater: %s.", err)
					panic(err)
				}
			}
		} else {
			if err := r.announcer.SetSubscriptions(); err != nil {
				log.Fatalf("Failed to subscribe to required bundle announcement topics: %s.", err);
			}
			r.announcer.SendAnnouncement();
		}
		if err := r.setSubscriptions(); err != nil {
			log.Errorf("Failed to set Relay subscriptions: %s.", err)
			panic(err)
		}
		if r.catalog.Len() > 0 {
			r.catalog.Reconnected()
		} else {
			log.Info("Loading bundle catalog.")
			r.requestBundles()
		}
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
		RelayConfig: r.config,
		Engines:     r.engines,
		Publisher:   r.conn,
		Catalog:     r.catalog,
		Topic:       topic,
		Payload:     message,
	}
	ctx := context.WithValue(context.Background(), "invoke", invoke)
	r.queue <- ctx
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
		log.Debug("Processing bundle catalog updates.")
		r.updateCatalog(tm.(*messages.ListBundlesResponseEnvelope))
	}
}

func (r *cogRelay) updateCatalog(envelope *messages.ListBundlesResponseEnvelope) {
	bundles := []*config.Bundle{}
	for _, b := range envelope.Bundles {
		b.ConfigFile.Version = fixBundleVersion(b.ConfigFile.Version)
		configFile := b.ConfigFile
		bundles = append(bundles, &configFile)
	}
	r.catalog.Replace(bundles)
	if r.catalog.IsChanged() {
		if err := r.refreshBundles(); err != nil {
			log.Errorf("Bundle catalog refresh failed: %s.", err)
		} else {
			log.Info("Changes to bundle catalog detected.")
			r.announcer.SendAnnouncement()
		}
	} else {
		log.Debug("Bundle catalog is unchanged.")
	}
	r.bundleTimer = time.AfterFunc(r.config.RefreshDuration(), r.scheduledBundleRefresh)
}

func (r *cogRelay) refreshBundles() error {
	var dockerEngine engines.Engine
	var err error
	if r.config.DockerEnabled() == true {
		dockerEngine, err = r.engines.GetEngine(engines.DockerEngineType)
		if err != nil {
			return err
		}
	}
	for _, name := range r.catalog.BundleNames() {
		if bundle := r.catalog.Find(name); bundle != nil {
			if bundle.NeedsRefresh() {
				if bundle.IsDocker() {
					if r.config.DockerEnabled() == false {
						log.Infof("Skipping Docker-based bundle %s %s.", bundle.Name, bundle.Version)
						bundle.SetAvailable(false)
						continue
					}
					avail, _ := dockerEngine.IsAvailable(bundle.Docker.Image, bundle.Docker.Tag)
					bundle.SetAvailable(avail)
				} else {
					engine, _ := r.engines.EngineForBundle(bundle)
					avail, _ := engine.IsAvailable(bundle.Name, bundle.Version)
					bundle.SetAvailable(avail)
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
	log.Debug("Refreshing command catalog.")
	return r.conn.Publish(infoTopic, raw)
}

func (r *cogRelay) scheduledBundleRefresh() {
	if err := r.requestBundles(); err != nil {
		log.Errorf("Scheduled bundle catalog refresh failed: %s.", err)
		r.bundleTimer = time.AfterFunc(r.config.RefreshDuration(), r.scheduledBundleRefresh)
	}
}

func (r *cogRelay) scheduledDockerCleanup() {
	cleaned := r.dockerEngine.Clean()
	container := "containers"
	if cleaned == 1 {
		container = "container"
	}
	if cleaned > 0 {
		log.Infof("Scheduled Docker clean up removed %d %s.", cleaned, container)
	}
	r.cleanTimer = time.AfterFunc(r.config.Docker.CleanDuration(), r.scheduledDockerCleanup)
}

func (r *cogRelay) makeConnOpts() bus.ConnectionOptions {
	connOpts := bus.ConnectionOptions{
		Userid:        r.config.ID,
		Password:      r.config.Cog.Token,
		Host:          r.config.Cog.Host,
		Port:          r.config.Cog.Port,
		SSLEnabled:    r.config.Cog.SSLEnabled,
		SSLCertPath:   r.config.Cog.SSLCertPath,
	}
	return connOpts
}


func fixBundleVersion(version string) string {
	if len(strings.Split(version, ".")) == 2 {
		return fmt.Sprintf("%s.0", version)
	}
	return version
}

func newWill(id string, replyTo string) string {
	announcement := messages.NewOfflineAnnouncement(id, replyTo)
	data, _ := json.Marshal(announcement)
	return string(data)
}
