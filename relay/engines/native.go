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

type NativeEngine struct {
	relayConfig config.Config
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

var notImplemented = errors.New("Not implemented")

func NewNativeEngine(relayConfig config.Config) (Engine, error) {
	return &NativeEngine{
		relayConfig: relayConfig,
		stdout:      new(bytes.Buffer),
		stderr:      new(bytes.Buffer),
	}, nil
}

func (ne *NativeEngine) IsAvailable(name string, meta string) (bool, error) {
	return false, notImplemented
}
func (ne *NativeEngine) IDForName(name string) (string, error) {
	return "", notImplemented
}

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
