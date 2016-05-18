package messages

import (
	"encoding/json"
	"errors"
)

var errorUnknownMessageType = errors.New("Unknown message type")

// ParseUntypedDirective inspects the JSON message
// and selects the appropriate struct to use
func ParseUntypedDirective(payload []byte) (interface{}, error) {
	var untypedPayload map[string]interface{}
	err := json.Unmarshal(payload, &untypedPayload)
	if err != nil {
		return nil, err
	}

	// ListBundlesResponseEnvelope
	if _, ok := untypedPayload["bundles"]; ok {
		result := &ListBundlesResponseEnvelope{}
		err = json.Unmarshal(payload, result)
		return result, err
	}

	return nil, errorUnknownMessageType
}
