package engines

import (
	"bytes"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"os/exec"
	"time"
)

// NativeEngine executes commands natively, that is directly,
// on the Relay host.
type NativeEngine struct {
	relayConfig config.Config
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

var errorNotImplemented = errors.New("Not implemented")
var errorDisabled = errors.New("Native execution engine is disabled.")

// NewNativeEngine constructs a new instance
func NewNativeEngine(relayConfig config.Config) (Engine, error) {
	if relayConfig.NativeEnabled() == true {
		return &NativeEngine{
			relayConfig: relayConfig,
			stdout:      new(bytes.Buffer),
			stderr:      new(bytes.Buffer),
		}, nil
	}
	return nil, errorDisabled
}

// IsAvailable required by engines.Engine interface
func (ne *NativeEngine) IsAvailable(name string, meta string) (bool, error) {
	return false, errorNotImplemented
}

// IDForName required by engines.Engine interface
func (ne *NativeEngine) IDForName(name string, meta string) (string, error) {
	return "", errorNotImplemented
}

// Execute runs a command invocation
func (ne *NativeEngine) Execute(request *messages.ExecutionRequest, bundle *config.Bundle) ([]byte, []byte, error) {
	emptyResult := []byte{}
	command := exec.Command(request.CommandName())
	command.Env = BuildEnvironment(*request, ne.relayConfig)
	input, _ := json.Marshal(request.CogEnv)
	command.Stdin = bytes.NewBuffer(input)
	command.Stdout = ne.stdout
	command.Stderr = ne.stderr
	start := time.Now()
	err := command.Run()
	finish := time.Now()
	log.Infof("Command %s ran for %f secs.", request.Command, finish.Sub(start).Seconds())
	if err != nil {
		return emptyResult, emptyResult, err
	}
	return ne.stdout.Bytes(), ne.stderr.Bytes(), nil
}

// Clean required by engines.Engine interface
func (ne *NativeEngine) Clean() int {
	return 0
}
