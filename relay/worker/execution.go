package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
)

func executeCommand(incoming *relay.Incoming) error {
	request := &messages.ExecutionRequest{}
	if err := json.Unmarshal(incoming.Payload, request); err != nil {
		return err
	}
	bundle := incoming.Relay.GetBundle(request.BundleName())
	if bundle == nil {
		return fmt.Errorf("Unknown command bundle %s", request.BundleName())
	}
	if bundle.IsDocker() == true {
		return executeDockerCommand(request, incoming, bundle)
	}
	return executeNativeCommand(request, incoming, bundle)
}

func executeDockerCommand(request *messages.ExecutionRequest, incoming *relay.Incoming, bundle *config.Bundle) error {
	engine, err := engines.NewDockerEngine(incoming.Relay.Config.Docker)
	if err == nil && engine == nil {
		log.Error("Docker engine is disabled.")
		return errors.New("Docker engine is disabled")
	}
	if err != nil {
		return err
	}
	commandOutput, commandErrors, err := engine.Execute(request, bundle)
	if err != nil {
		return err
	}
	log.Infof("Command returned:\n%s\n%s", string(commandOutput), string(commandErrors))
	return nil
}

func executeNativeCommand(request *messages.ExecutionRequest, incoming *relay.Incoming, bundle *config.Bundle) error {
	return nil
}
