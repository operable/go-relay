package relay

import (
	"container/list"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
	"sync"
)

// ControlCommand are async signals sent to a running Relay
type ControlCommand int

// Allowed commands
const (
	RelayStop ControlCommand = iota
	RelayRestart
	RelayUpdateBundles
	RelayUpdateBundlesDone
)

// State describes Relay's various runtime states
type State int

// Runtime states
const (
	RelayStopped State = iota
	RelayStarting
	RelayUpdatingBundles
	RelayReady
)

// Worker pulls work items from a Relay's work queue
type Worker func(*Queue, sync.WaitGroup)

// Incoming request or directive
type Incoming struct {
	Relay   *Relay
	Topic   string
	Payload []byte
}

// Relay is a single instance of a Relay
type Relay struct {
	Config        *config.Config
	Bus           bus.MessageBus
	bundleLock    sync.RWMutex
	bundles       map[string]*config.Bundle
	fetchedImages *list.List
	workQueue     *Queue
	coordinator   sync.WaitGroup
	control       chan ControlCommand
	state         State
}

// New creates a new Relay instance with the specified config
func New(relayConfig *config.Config) *Relay {
	return &Relay{
		Config:        relayConfig,
		bundles:       make(map[string]*config.Bundle),
		fetchedImages: list.New(),
		// Create work queue with some burstable capacity
		workQueue: NewQueue(relayConfig.MaxConcurrent * 2),
		control:   make(chan ControlCommand),
		state:     RelayStopped,
	}
}

// Start initializes a Relay. Returns an error
// if Docker config fails verification or if it
// can't connect to the upstream Cog instance.
func (r *Relay) Start(worker Worker) error {
	if err := r.verifyDockerConfig(); err != nil {
		return err
	}
	r.state = RelayStarting
	r.startWorkers(worker)
	err := r.connectToCog()
	if err != nil {
		r.Stop()
		return err
	}
	go r.runLoop()
	return nil
}

// Stop a running relay
func (r *Relay) Stop() {
	if r.state != RelayStopped {
		if r.Bus != nil {
			r.Bus.Halt()
		}
		r.workQueue.Stop()
		r.coordinator.Wait()
		r.state = RelayStopped
	}
}

// UpdateBundles causes a Relay to ask Cog
// for its bundle assignments
func (r *Relay) UpdateBundles() bool {
	if r.state != RelayStarting {
		return false
	}
	r.control <- RelayUpdateBundles
	return true
}

// FinishedUpdateBundles is used by worker processes to
// signal when the a bundle refresh is complete.
func (r *Relay) FinishedUpdateBundles() bool {
	if r.state != RelayUpdatingBundles {
		return false
	}
	r.control <- RelayUpdateBundlesDone
	return true
}

// GetBundle returns the named config.Bundle or nil
func (r *Relay) GetBundle(name string) *config.Bundle {
	r.bundleLock.RLock()
	defer r.bundleLock.RUnlock()
	return r.bundles[name]
}

// StoreBundle stores a bundle config
func (r *Relay) StoreBundle(bundle *config.Bundle) {
	r.bundleLock.Lock()
	defer r.bundleLock.Unlock()
	r.bundles[bundle.Name] = bundle
}

func (r *Relay) startWorkers(worker Worker) {
	for i := 0; i < r.Config.MaxConcurrent; i++ {
		go func() {
			worker(r.workQueue, r.coordinator)
		}()
	}
	log.Infof("Started %d workers.", r.Config.MaxConcurrent)

}

func (r *Relay) connectToCog() error {
	// Handler func used for both message types
	handler := func(bus bus.MessageBus, topic string, payload []byte) {
		r.handleMessage(topic, payload)
	}

	// Connect to Cog
	subs := bus.Subscriptions{
		CommandHandler:   handler,
		ExecutionHandler: handler,
	}
	link, err := bus.NewLink(r.Config.ID, r.Config.Cog, subs, r.coordinator)
	if err != nil {
		log.Errorf("Error connecting to Cog: %s.", err)
		return err
	}

	log.Infof("Connected to Cog host %s.", r.Config.Cog.Host)
	err = link.Run()
	if err != nil {
		log.Errorf("Error subscribing to message topics: %s.", err)
		return err
	}
	r.Bus = link
	return nil
}

func (r *Relay) handleMessage(topic string, payload []byte) {
	message := &Incoming{
		Relay:   r,
		Topic:   topic,
		Payload: payload,
	}
	ctx := context.WithValue(context.Background(), "message", message)
	r.workQueue.Enqueue(ctx)
}

func (r *Relay) verifyDockerConfig() error {
	if r.Config.DockerDisabled == false {
		if err := engines.VerifyDockerConfig(r.Config.Docker); err != nil {
			log.Errorf("Error verifying Docker configuration: %s.", err)
			return err
		}
		log.Infof("Docker configuration verified.")
	} else {
		log.Infof("Docker support disabled.")
	}
	return nil
}

func (r *Relay) runLoop() {
	for {
		switch <-r.control {
		case RelayStop:
			r.handleStopCommand()
			return
		case RelayRestart:
			r.handleRestartCommand()
			return
		case RelayUpdateBundles:
			r.handleUpdateBundlesCommand()
		case RelayUpdateBundlesDone:
			r.handleUpdateBundlesDone()
		}
	}
}

func (r *Relay) handleStopCommand() {
	r.Stop()
}

func (r *Relay) handleRestartCommand() {
}

func (r *Relay) handleUpdateBundlesDone() {
	if r.state == RelayUpdatingBundles {
		log.Info("Bundle refresh complete.")
		log.Infof("Relay %s ready.", r.Config.ID)
		r.state = RelayReady
	} else {
		r.logBadState("handleUpdatesBundleDone", RelayUpdatingBundles)
	}
}

func (r *Relay) handleUpdateBundlesCommand() {
	if r.state == RelayStarting {
		msg := messages.ListBundlesEnvelope{
			ListBundles: &messages.ListBundlesMessage{
				RelayID: r.Config.ID,
				ReplyTo: r.Bus.DirectiveReplyTo(),
			},
		}
		raw, _ := json.Marshal(&msg)
		log.Info("Refreshing command bundles.")
		r.Bus.Publish("bot/relays/info", raw)
		r.state = RelayUpdatingBundles
	} else {
		r.logBadState("handleUpdateBundlesCommand", RelayStarting)
	}
}

func (r *Relay) logBadState(name string, required State) {
	log.Errorf("%s requires relay state %d: %d.", name, required, r.state)
}
