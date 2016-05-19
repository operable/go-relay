package config

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/go-yaml/yaml"
	"strings"
)

// RawConfig is unparsed config.Config data
type RawConfig []byte

// IsEmpty returns true if RawConfig's backing byte array is empty
func (rc RawConfig) IsEmpty() bool {
	return len(rc) == 0
}

// Parse parses Relay's raw config and then produces a final
// version incorporating default values and environment variable values.
// Finalization proceeds using the following steps:
// 1. Parse YAML and populate new Config struct instance with values
// 2. Set default values on unassigned fields (for fields with defaults)
// 3. Apply environment variable overrides
// 4. Validate the finalized config
func (rc RawConfig) Parse() (*Config, error) {
	var config Config
	if rc.IsEmpty() {
		config.Version = RelayConfigVersion
	} else {
		err := yaml.Unmarshal(rc, &config)
		if err != nil {
			return nil, err
		}
		if config.Version != RelayConfigVersion {
			return nil, errorWrongConfigVersion
		}
	}
	config.populate()
	govalidator.TagMap["hostorip"] = govalidator.Validator(func(value string) bool {
		return govalidator.IsHost(value)
	})
	govalidator.TagMap["dockersocket"] = govalidator.Validator(func(value string) bool {
		return strings.HasPrefix(value, "unix://") || strings.HasPrefix(value, "tcp://")
	})
	_, err := govalidator.ValidateStruct(config)
	if err == nil && config.Docker != nil {
		_, err = govalidator.ValidateStruct(config.Docker)
		return &config, err
	}
	fmt.Printf("log level: %s", config.LogLevel)
	return &config, err
}
