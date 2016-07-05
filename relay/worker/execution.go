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
	"github.com/operable/go-relay/relay/util"
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
			decoder = util.NewJSONDecoder(bufferedReader)
		} else {
			bufferedReader.Reset(bytes.NewReader(invoke.Payload))
		}
		executeStage(decoder, invoke)
	}
}

func executeStage(decoder *json.Decoder, invoke *CommandInvocation) {
	stageReq := &messages.PipelineStageRequest{}
	if err := decoder.Decode(stageReq); err != nil {
		log.Errorf("Ignoring malformed pipeline stage execution request: %s.", err)
		return
	}
	stageReq.Parse()
	bundle := invoke.Catalog.Find(stageReq.BundleName())
	response := &messages.PipelineStageResponse{
		Responses: []messages.ExecutionResponse{},
	}
	if bundle == nil {
		addError(response, fmt.Sprintf("Unknown command bundle %s", stageReq.BundleName()))
	} else {
		engine, err := invoke.Engines.EngineForBundle(bundle)
		if err != nil {
			addError(response, fmt.Sprintf("%s", err))
		} else {
			env, err := engine.NewEnvironment(stageReq.PipelineID(), bundle)
			if err != nil {
				addError(response, fmt.Sprintf("%s", err))
			} else {
				defer engine.ReleaseEnvironment(stageReq.PipelineID(), bundle, env)
				for _, req := range stageReq.Requests {
					resp := &messages.ExecutionResponse{}
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
					circuitRequest, foundDynamicConfig := stageReq.ToCircuitRequest(&req, bundle, invoke.RelayConfig, hasDynamicConfig)
					if foundDynamicConfig == false && hasDynamicConfig == true {
						userData["dynamic-config"] = false
						env.SetUserData(userData)
					}
					result, err := env.Run(*circuitRequest)
					parseOutput(result, err, resp, *stageReq, req)
					response.Responses = append(response.Responses, *resp)
					if err != nil {
						break
					}
				}
			}
		}
	}
	responseBytes, _ := json.Marshal(response)
	invoke.Publisher.Publish(stageReq.ReplyTo, responseBytes)
}

func addError(stageResponse *messages.PipelineStageResponse, message string) {
	resp := messages.ExecutionResponse{
		Status:        "error",
		StatusMessage: message,
	}
	stageResponse.Responses = append(stageResponse.Responses, resp)
}
