package relay

import (
	"container/list"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/bundle"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
	"strings"
	"sync"
	"time"
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

var errorNoExecutionEngines = errors.New("Invalid Relay configuration detected. At least one execution engine must be enabled.")

// Worker pulls work items from a Relay's work queue
type Worker func(Queue, sync.WaitGroup)

// Incoming request or directive
type Incoming struct {
	Relay       *Relay
	IsExecution bool
	Topic       string
	Payload     []byte
}

// Relay is a single instance of a Relay
type Relay struct {
	Config        *config.Config
	Bus           bus.MessageBus
	announcer     Announcer
	bundles       *bundle.Catalog
	fetchedImages *list.List
	workQueue     Queue
	worker        Worker
	refreshTimer  *time.Timer
	dockerTimer   *time.Timer
	hasStarted    bool
	coordinator   sync.WaitGroup
	control       chan ControlCommand
	state         State
}

// New creates a new Relay instance with the specified config
func New(relayConfig *config.Config) *Relay {
	return &Relay{
		Config:        relayConfig,
		bundles:       bundle.NewCatalog(),
		fetchedImages: list.New(),
		workQueue:     NewQueue(uint(relayConfig.MaxConcurrent)),
		control:       make(chan ControlCommand, 2),
		state:         RelayStopped,
	}
}

// Start initializes a Relay. Returns an error
// if execution engines or Docker config fails verification
func (r *Relay) Start(worker Worker) error {
	if err := r.verifyEnabledExecutionEngines(); err != nil {
		return err
	}
	if err := r.verifyDockerConfig(); err != nil {
		return err
	}
	r.state = RelayStarting
	r.startWorkers(worker)
	r.connectToCog()
	r.worker = worker
	go r.runLoop()
	return nil
}

// Stop a running relay
func (r *Relay) Stop() {
	if r.state != RelayStopped {
		r.stopTimers()
		if r.announcer != nil {
			r.announcer.Halt()
		}
		if r.Bus != nil {
			r.Bus.Halt()
		}
		r.workQueue.Stop(true)
		r.control <- RelayStop
		r.coordinator.Wait()
		r.state = RelayStopped
	}
}

// UpdateBundles causes a Relay to ask Cog
// for its bundle assignments
func (r *Relay) UpdateBundles() bool {
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
	return r.bundles.FindLatest(name)
}

// UpdateBundleList atomically replaces the existing master bundle list
// with a new one
func (r *Relay) UpdateBundleList(bundles map[string]*config.Bundle) {
	batch := make([]*config.Bundle, len(bundles))
	i := 0
	for _, b := range bundles {
		batch[i] = b
		i++
	}
	r.bundles.AddBatch(batch)
}

// GetBundles returns a list of bundles known by a Relay
func (r *Relay) GetBundles() []config.Bundle {
	names := r.bundles.BundleNames()
	retval := make([]config.Bundle, len(names))
	i := 0
	for _, name := range names {
		bundle := r.bundles.FindLatest(name)
		retval[i] = *bundle
		i++
	}
	return retval
}

// BundleNames returns list of bundle names known by a Relay
func (r *Relay) BundleNames() []string {
	return r.bundles.BundleNames()
}

func (r *Relay) startWorkers(worker Worker) {
	workerCount := r.Config.MaxConcurrent + 2
	for i := 0; i < workerCount; i++ {
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
	handlers := bus.Handlers{
		CommandHandler:    handler,
		ExecutionHandler:  handler,
		DisconnectHandler: r.disconnected,
	}
	link, err := bus.NewLink(r.Config.ID, r.Config.Cog, handlers, r.coordinator)
	if err != nil {
		log.Errorf("Error connecting to Cog: %s.", err)
		return err
	}
	err = link.Run()
	if err != nil {
		log.Errorf("Error connecting to Cog: %s.", err)
		return err
	}
	r.Bus = link
	if r.announcer == nil {
		r.announcer, _ = NewAnnouncer(r, r.coordinator)
	}
	if err := r.announcer.Run(); err != nil {
		r.Bus.Halt()
		return err
	}
	log.Infof("Connected to Cog host %s.", r.Config.Cog.Host)
	return nil
}

func (r *Relay) disconnected(err error) {
	log.Errorf("Relay %s disconnected due to error: %s.", r.Config.ID, err)
	r.control <- RelayRestart
}

func (r *Relay) handleMessage(topic string, payload []byte) {
	incoming := &Incoming{
		Relay:       r,
		Topic:       topic,
		IsExecution: strings.HasPrefix(topic, "/bot/commands/"),
		Payload:     payload,
	}
	ctx := context.WithValue(context.Background(), "incoming", incoming)
	r.workQueue.Enqueue(ctx)
}

func (r *Relay) verifyDockerConfig() error {
	if r.Config.DockerEnabled() == true {
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

func (r *Relay) verifyEnabledExecutionEngines() error {
	if r.Config.DockerEnabled() == false && r.Config.NativeEnabled() == false {
		log.Errorf("%s", errorNoExecutionEngines)
		return errorNoExecutionEngines
	}
	if r.Config.DockerEnabled() == true {
		log.Info("Docker execution engine enabled.")
	}
	if r.Config.NativeEnabled() == true {
		log.Info("Native execution engine enabled.")
	}
	return nil
}

func (r *Relay) runLoop() {
	r.coordinator.Add(1)
	defer r.coordinator.Done()
	for {
		switch <-r.control {
		case RelayStop:
			return
		case RelayRestart:
			r.handleRestartCommand()
		case RelayUpdateBundles:
			r.handleUpdateBundlesCommand()
		case RelayUpdateBundlesDone:
			r.handleUpdateBundlesDone()
		}
	}
}

func (r *Relay) handleRestartCommand() {
	if r.Bus != nil {
		r.Bus.Halt()
	}
	if r.announcer != nil {
		r.announcer.Halt()
	}
	r.workQueue.Stop(true)
	r.coordinator.Done()
	r.coordinator.Wait()
	r.state = RelayStopped

	log.Infof("Relay %s restarting.", r.Config.ID)
	r.bundles = bundle.NewCatalog()
	r.coordinator.Add(1)
	r.state = RelayStarting
	r.workQueue.Start()
	r.startWorkers(r.worker)
	if err := r.connectToCog(); err != nil {
		log.Fatalf("Restarting Relay %s failed: %s.", r.Config.ID, err)
		panic(err)
	}
	r.control <- RelayUpdateBundles
}

func (r *Relay) handleUpdateBundlesDone() {
	if r.state == RelayUpdatingBundles {
		if r.bundles.ShouldAnnounce() {
			r.announceBundles()
		}
		log.Info("Bundle refresh complete.")
		if r.hasStarted == false {
			log.Infof("Relay %s ready.", r.Config.ID)
			r.hasStarted = true
		}
		r.state = RelayReady
	} else {
		r.logBadState("handleUpdatesBundleDone", RelayUpdatingBundles)
	}
}

func (r *Relay) handleUpdateBundlesCommand() {
	if r.state == RelayStarting {
		log.Infof("Refreshing bundles and related assets every %s.", r.Config.RefreshDuration())
		r.setRefreshTimer()
		if r.Config.DockerEnabled() == true {
			log.Infof("Cleaning up expired Docker assets every %s.", r.Config.Docker.CleanDuration())
			r.setDockerTimer()
		}
	}
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
}

func (r *Relay) logBadState(name string, required State) {
	log.Errorf("%s requires relay state %d: %d.", name, required, r.state)
}

func (r *Relay) announceBundles() {
	r.announcer.SendAnnouncement()
}

func (r *Relay) stopTimers() {
	if r.refreshTimer != nil {
		r.refreshTimer.Stop()
		r.refreshTimer = nil
	}
	if r.dockerTimer != nil {
		r.dockerTimer.Stop()
		r.dockerTimer = nil
	}
}

func (r *Relay) setRefreshTimer() {
	r.refreshTimer = time.AfterFunc(r.Config.RefreshDuration(), r.triggerRefresh)
}

func (r *Relay) setDockerTimer() {
	if r.Config.DockerEnabled() == false {
		return
	}
	r.dockerTimer = time.AfterFunc(r.Config.Docker.CleanDuration(), r.triggerDockerClean)
}

func (r *Relay) triggerRefresh() {
	r.UpdateBundles()
	r.setRefreshTimer()
}

func (r *Relay) triggerDockerClean() {
	if r.Config != nil {
		dockerEngine, err := engines.NewDockerEngine(*r.Config)
		if err != nil {
			panic(err)
		}
		count := dockerEngine.Clean()
		switch count {
		case 0:
			break
		case 1:
			log.Info("Removed 1 dead Docker container.")
		default:
			log.Infof("Removed %d dead Docker containers.", count)
		}
	}
	r.setDockerTimer()
}
