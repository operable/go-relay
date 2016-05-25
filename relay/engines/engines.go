package engines

import (
	"errors"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines/exec"
)

// EngineType is an enum describing the various engine types
// supported.
type EngineType int

const (
	// DockerEngineType constant
	DockerEngineType EngineType = iota
	// NativeEngineType constant
	NativeEngineType
)

// ErrDockerDisabled indicates the Docker engine is disabled and
// therefore unavailable for use.
var ErrDockerDisabled = errors.New("Docker engine is disabled")

// Engine defines the execution engine interface
type Engine interface {
	Init() error
	IsAvailable(name string, meta string) (bool, error)
	NewEnvironment(bundle *config.Bundle) (exec.Environment, error)
	ReleaseEnvironment(exec.Environment)
	Clean() int
}

// Engines knows how to create engines based on bundle type
type Engines struct {
	relayConfig *config.Config
}

// NewEngines constructs a new Engines instance
func NewEngines(relayConfig *config.Config) *Engines {
	return &Engines{
		relayConfig: relayConfig,
	}
}

// EngineForBundle returns the correct engine for a given
// bundle type.
func (e *Engines) EngineForBundle(bundle *config.Bundle) (Engine, error) {
	if bundle.IsDocker() {
		return e.GetEngine(DockerEngineType)
	}
	return e.GetEngine(NativeEngineType)
}

// GetEngine returns the specified engine (if available)
func (e *Engines) GetEngine(engineType EngineType) (Engine, error) {
	if engineType == DockerEngineType {
		if e.relayConfig.DockerEnabled() {
			return NewDockerEngine(e.relayConfig)
		}
		return nil, ErrDockerDisabled
	}
	return NewNativeEngine(e.relayConfig)
}
