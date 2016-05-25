package exec

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"os/exec"
	"time"
)

// NativeEnvironment is an execution environment which runs
// inside of a native OS process directly on the Relay host.
type NativeEnvironment struct {
	relayConfig *config.Config
	bundle      *config.Bundle
}

// NewNativeEnvironment creates a new NativeEnvironment instance
func NewNativeEnvironment(relayConfig *config.Config, bundle *config.Bundle) (Environment, error) {
	return &NativeEnvironment{
		relayConfig: relayConfig,
		bundle:      bundle,
	}, nil
}

func (ne *NativeEnvironment) Terminate(kill bool) {}

func (ne *NativeEnvironment) BundleName() string {
	return ne.bundle.Name
}

// Execute is required by the exec.Environment interface
func (ne *NativeEnvironment) Execute(request *messages.ExecutionRequest) ([]byte, []byte, error) {
	if bundleCommand := ne.bundle.Commands[request.CommandName()]; bundleCommand != nil {
		command := exec.Command(bundleCommand.Executable)
		command.Env = BuildCallingEnvironment(request, ne.relayConfig)
		input, _ := json.Marshal(request.CogEnv)
		stdout := bytes.NewBuffer([]byte{})
		stderr := bytes.NewBuffer([]byte{})
		command.Stdin = bytes.NewBuffer(input)
		command.Stdout = stdout
		command.Stderr = stderr
		start := time.Now()
		err := command.Run()
		finish := time.Now()
		log.Infof("Command %s ran for %f secs.", request.Command, finish.Sub(start).Seconds())
		return stdout.Bytes(), stderr.Bytes(), err
	}
	return EmptyResult, EmptyResult, ErrUnknownCommand
}
