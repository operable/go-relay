package worker

import (
	"github.com/operable/circuit-driver/api"
	"github.com/operable/go-relay/relay/messages"
)

// OutputParser parses logging directives and content emitted by commands
type OutputParser interface {
	Parse(api.ExecResult, messages.ExecutionRequest, error) *messages.ExecutionResponse
}
