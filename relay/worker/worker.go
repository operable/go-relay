package worker

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
	"sync"
)

// RunWorker is a the logic loop for a request worker
func RunWorker(workQueue *relay.Queue, coordinator sync.WaitGroup) {
	coordinator.Add(1)
	defer coordinator.Done()
	for thing := workQueue.Dequeue(); thing != nil; thing = workQueue.Dequeue() {
		// Convert dequeued thing to context
		ctx, ok := thing.(context.Context)

		if ok == false {
			log.Error("Dropping improperly queued request.")
			continue
		}

		// Extract message and parse payload
		msg := ctx.Value("message").(*relay.Incoming)

		result, err := parsePayload(msg.Payload)
		if err != nil {
			log.Errorf("Failed to parse payload '%s': %s.", string(msg.Payload), err)
			continue
		}

		// Dispatch on mesasge type
		switch result.(type) {
		case *messages.ListBundlesResponseEnvelope:
			updateBundles(ctx, result.(*messages.ListBundlesResponseEnvelope))
		}
	}
}

func parsePayload(payload []byte) (interface{}, error) {
	var untypedPayload map[string]interface{}
	err := json.Unmarshal(payload, &untypedPayload)
	if err != nil {
		return nil, err
	}
	// ListBundlesResponseEnvelope
	if _, ok := untypedPayload["bundles"]; ok {
		result := &messages.ListBundlesResponseEnvelope{}
		err = json.Unmarshal(payload, result)
		return result, err
	}
	return nil, fmt.Errorf("Unknown message type for payload '%s'.", string(payload))
}
