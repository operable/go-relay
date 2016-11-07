package engines

import (
	"errors"
	"github.com/operable/circuit"
	"github.com/operable/go-relay/relay/config"
)

// NativeEngine executes commands natively, that is directly,
// on the Relay host.
type NativeEngine struct {
	relayConfig *config.Config
}

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

// Init required by engines.Engine interface
func (ne *NativeEngine) Init() error {
	return nil
}

// IsAvailable required by engines.Engine interface
func (ne *NativeEngine) IsAvailable(name string, version string) (bool, error) {
	return true, nil
}

// NewEnvironment is required by the engines.Engine interface
func (ne *NativeEngine) NewEnvironment(pipelineID string, bundle *config.Bundle) (circuit.Environment, error) {
	options := circuit.CreateEnvironmentOptions{}
	options.Kind = circuit.NativeKind
	options.Bundle = bundle.Name
	return circuit.CreateEnvironment(options)
}

// ReleaseEnvironment is required by the engines.Engine interface
func (ne *NativeEngine) ReleaseEnvironment(pipelineID string, bundle *config.Bundle, env circuit.Environment) {
	env.Shutdown()
}

// Clean required by engines.Engine interface
func (ne *NativeEngine) Clean() int {
	return 0
}
