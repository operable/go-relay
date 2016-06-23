package worker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/circuit"
	"github.com/operable/go-relay/relay/bundle"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
)

// CommandInvocation request
type CommandInvocation struct {
	RelayConfig *config.Config
	Publisher   bus.MessagePublisher
	Catalog     *bundle.Catalog
	Engines     *engines.Engines
	Topic       string
	Payload     []byte
	Shutdown    bool
}

// ExecutionWorker is the entry point for command execution
// goroutines.
func ExecutionWorker(queue chan interface{}) {
	var decoder *json.Decoder
	var bufferedReader *bufio.Reader
	for {
		thing := <-queue
		// Convert dequeued thing to context
		ctx, ok := thing.(context.Context)

		if ok == false {
			log.Error("Dropping improperly queued request.")
			continue
		}
		invoke := ctx.Value("invoke").(*CommandInvocation)
		if bufferedReader == nil {
			bufferedReader = bufio.NewReader(bytes.NewReader(invoke.Payload))
			decoder = json.NewDecoder(bufferedReader)
		} else {
			bufferedReader.Reset(bytes.NewReader(invoke.Payload))
		}
		executeCommand(decoder, invoke)
	}
}

func executeCommand(decoder *json.Decoder, invoke *CommandInvocation) {
	request := &messages.ExecutionRequest{}

	if err := decoder.Decode(request); err != nil {
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
		engine, err := invoke.Engines.EngineForBundle(bundle)
		if err != nil {
			setError(response, err)
		} else {
			env, err := engine.NewEnvironment(request.PipelineID(), bundle)
			if err != nil {
				setError(response, err)
			} else {
				userData, _ := env.GetUserData()
				if userData == nil {
					userData = make(circuit.EnvironmentUserData)
				}
				hasDynamicConfig := true
				value, keyPresent := userData["dynamic-config"]
				if keyPresent == false {
					value = true
				}
				hasDynamicConfig = value.(bool)
				circuitRequest, foundDynamicConfig := request.ToCircuitRequest(bundle, invoke.RelayConfig, hasDynamicConfig)
				if foundDynamicConfig == false {
					userData["dynamic-config"] = false
					env.SetUserData(userData)
				}
				result, err := env.Run(circuitRequest)
				engine.ReleaseEnvironment(request.PipelineID(), bundle, env)
				parseOutput(result, err, response, *request)
			}
		}
	}
	responseBytes, _ := json.Marshal(response)
	invoke.Publisher.Publish(request.ReplyTo, responseBytes)
}

func setError(resp *messages.ExecutionResponse, err error) {
	resp.Status = "error"
	resp.StatusMessage = fmt.Sprintf("%s", err)
}
