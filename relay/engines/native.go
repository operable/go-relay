package engines

import (
	"errors"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines/exec"
)

// NativeEngine executes commands natively, that is directly,
// on the Relay host.
type NativeEngine struct {
	relayConfig *config.Config
}

var errorNotImplemented = errors.New("Not implemented")
var errorDisabled = errors.New("Native execution engine is disabled.")
var errorUnknownCommand = errors.New("Unknown command")

// NewNativeEngine constructs a new instance
func NewNativeEngine(relayConfig *config.Config) (Engine, error) {
	if relayConfig.NativeEnabled() == true {
		return &NativeEngine{
			relayConfig: relayConfig,
		}, nil
	}
	return nil, errorDisabled
}

// IsAvailable required by engines.Engine interface
func (ne *NativeEngine) IsAvailable(name string, meta string) (bool, error) {
	return false, errorNotImplemented
}

// NewEnvironment is required by the engines.Engine interface
func (ne *NativeEngine) NewEnvironment(bundle *config.Bundle) (exec.Environment, error) {
	return exec.NewNativeEnvironment(ne.relayConfig, bundle)
}

// Clean required by engines.Engine interface
func (ne *NativeEngine) Clean() int {
	return 0
}
