package exec

import (
	"errors"
	"github.com/operable/go-relay/relay/messages"
)

// Environment is an execution environment managed by an Engine.
// Environments are used to execute commands.
type Environment interface {
	BundleName() string
	Execute(request *messages.ExecutionRequest) ([]byte, []byte, error)
	Terminate(kill bool)
}

// EmptyResult is used to signify empty normal or error output
var EmptyResult []byte

// ErrUnknownCommand indicates an engine attempted to execute
// an unknown bundle command. In practice, this should rarely happen.
var ErrUnknownCommand = errors.New("Unknown command")
