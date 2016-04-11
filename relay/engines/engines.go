package engines

import (
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
)

// Engine defines the execution engine interface
type Engine interface {
	IsAvailable(name string, meta string) (bool, error)
	Execute(request *messages.ExecutionRequest, bundle *config.Bundle) ([]byte, []byte, error)
	IDForName(name string, meta string) (string, error)
	Clean() int
}

// Placeholder for empty results
var emptyResult = []byte{}
