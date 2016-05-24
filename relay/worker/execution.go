package worker

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
	"golang.org/x/net/context"
	"time"
)

// CommandInvocation request
type CommandInvocation struct {
	RelayConfig config.Config
	Publisher   bus.MessagePublisher
	Catalog     bundle.Catalog
	Topic       string
	Payload     []byte
	Shutdown    bool
}

// ExecutionWorker is the entry point for command execution
// goroutines.
func ExecutionWorker(workQueue util.Queue) {
	for {
		thing, err := workQueue.Dequeue()
		if err != nil {
			if workQueue.IsStopped() {
				time.Sleep(time.Duration(50) * time.Millisecond)
				continue
			}
			log.Errorf("Failed to dequeue request item: %s.", err)
			return
		}
		// Convert dequeued thing to context
		ctx, ok := thing.(context.Context)

		if ok == false {
			log.Error("Dropping improperly queued request.")
			continue
		}

		invoke := ctx.Value("invoke").(*CommandInvocation)
		executeCommand(invoke)
	}
}

func engineForBundle(bundle config.Bundle, config config.Config) (engines.Engine, error) {
	if bundle.IsDocker() == true {
		return engines.NewDockerEngine(config)
	}
	return engines.NewNativeEngine(config)
}

func executeCommand(invoke *CommandInvocation) {
	request := &messages.ExecutionRequest{}
	if err := json.Unmarshal(invoke.Payload, request); err != nil {
		log.Errorf("Ignoring malformed execution request: %s.", err)
		return
	}
	request.Parse()
	bundle := invoke.Catalog.Find(request.BundleName())
	response := &messages.ExecutionResponse{}
	if bundle == nil {
		response.Status = "error"
		response.StatusMessage = fmt.Sprintf("Unknown command bundle %s", request.BundleName())
	} else {
		engine, err := engineForBundle(*bundle, invoke.RelayConfig)
		if err != nil {
			response.Status = "error"
			response.StatusMessage = fmt.Sprintf("%s", err)
		} else {
			commandOutput, commandErrors, err := engine.Execute(request, bundle)
			parseOutput(commandOutput, commandErrors, err, response, *request)
		}
	}
	responseBytes, _ := json.Marshal(response)
	invoke.Publisher.Publish(request.ReplyTo, responseBytes)
}
