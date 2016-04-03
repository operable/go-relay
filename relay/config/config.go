package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-yaml/yaml"
)

const (
	// RelayConfigVersion describes the compatible Relay config file format version
	RelayConfigVersion = 1
)

// Config is the top level struct for all Relay configuration
type Config struct {
	Version        int            `yaml:"version" valid:"int64,required"`
	ID             string         `yaml:"id" env:"RELAY_ID" valid:"uuid,required"`
	MaxConcurrent  int            `yaml:"max_concurrent" env:"RELAY_MAX_CONCURRENT" valid:"int64,required" default:"16"`
	LogLevel       string         `yaml:"log_level" env:"RELAY_LOG_LEVEL" valid:"required" default:"info"`
	LogJSON        bool           `yaml:"log_json" env:"RELAY_LOG_JSON" valid:"bool" default:"false"`
	LogPath        string         `yaml:"log_path" env:"RELAY_LOG_PATH" valid:"required" default:"stdout"`
	Cog            *CogInfo       `yaml:"cog" valid:"required"`
	DockerDisabled bool           `yaml:"disable_docker" env:"RELAY_DISABLE_DOCKER" valid:"bool" default:"false"`
	Docker         *DockerInfo    `yaml:"docker" valid:"-"`
	Execution      *ExecutionInfo `yaml:"execution" valid:"required"`
}

// CogInfo contains information required to connect to an upstream Cog host
type CogInfo struct {
	Host  string `yaml:"host" env:"RELAY_COG_HOST" valid:"hostorip,required" default:"127.0.0.1"`
	Port  int    `yaml:"port" env:"RELAY_COG_PORT" valid:"int64,required" default:"1883"`
	Token string `yaml:"token" env:"RELAY_COG_TOKEN" valid:"required"`
}

// DockerInfo contains information required to interact with dockerd and external Docker registries
type DockerInfo struct {
	UseEnv           bool   `yaml:"use_env" env:"RELAY_DOCKER_USE_ENV" valid:"-" default:"false"`
	SocketPath       string `yaml:"socket_path" env:"RELAY_DOCKER_SOCKET_PATH" valid:"dockersocket,required" default:"unix:///var/run/docker.sock"`
	RegistryHost     string `yaml:"registry_host" env:"RELAY_DOCKER_REGISTRY_HOST" valid:"host,required" default:"hub.docker.com"`
	RegistryUser     string `yaml:"registry_user" env:"RELAY_DOCKER_REGISTRY_USER" valid:"-"`
	RegistryPassword string `yaml:"registry_password" env:"RELAY_DOCKER_REGISTRY_PASSWORD" valid:"-"`
}

// ExecutionInfo applies to every container for a given Relay host
type ExecutionInfo struct {
	CPUShares int64    `yaml:"cpu_shares" env:"RELAY_CONTAINER_CPUSHARES" valid:"int64"`
	CPUSet    string   `yaml:"cpu_set" env:"RELAY_CONTAINER_CPUSET"`
	ExtraEnv  []string `yaml:"env" env:"RELAY_CONTAINER_ENV"`
}

func applyDefaults(config *Config) {
	setDefaultValues(config)
	if config.Cog != nil {
		setDefaultValues(config.Cog)
	} else {
		cog := &CogInfo{}
		if setDefaultValues(cog) == true {
			config.Cog = cog
		}
	}
	if config.Docker != nil {
		setDefaultValues(config.Docker)
	} else {
		docker := &DockerInfo{}
		if setDefaultValues(docker) == true {
			config.Docker = docker
		}
	}
	if config.Execution != nil {
		setDefaultValues(config.Execution)
	} else {
		execution := &ExecutionInfo{}
		if setDefaultValues(execution) == true {
			config.Execution = execution
		}
	}
}

func setDefaultValues(config interface{}) bool {
	t := reflect.ValueOf(config).Elem()
	configType := t.Type()
	updatedConfig := false
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fdef := configType.Field(i)
		defaultValue := fdef.Tag.Get("default")
		if defaultValue != "" {
			currentValue := f.Interface()
			switch f.Type().Kind() {
			case reflect.Int:
				if currentValue == 0 {
					if newValue, err := strconv.ParseInt(defaultValue, 10, 64); err == nil {
						t.Field(i).SetInt(newValue)
						updatedConfig = true
					} else {
						panic(err)
					}
				}
			case reflect.Bool:
				if newValue, err := strconv.ParseBool(defaultValue); err == nil {
					if currentValue == false && newValue == true {
						t.Field(i).SetBool(newValue)
						updatedConfig = true
					}
				} else {
					panic(err)
				}
			case reflect.String:
				if currentValue == "" {
					t.Field(i).SetString(defaultValue)
					updatedConfig = true
				}
			}
		}
	}
	return updatedConfig
}

func applyEnvVars(config *Config) {
	setEnvVars(config)
	if config.Cog != nil {
		setEnvVars(config.Cog)
	}
	if config.Docker != nil {
		setEnvVars(config.Docker)
	}
	if config.Execution != nil {
		setEnvVars(config.Execution)
	}
}

func envVarForTag(varName string) string {
	if varName == "" {
		return ""
	}
	return os.Getenv(varName)
}

func setEnvVars(config interface{}) bool {
	t := reflect.ValueOf(config).Elem()
	configType := t.Type()
	updatedConfig := false
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fdef := configType.Field(i)
		envValue := envVarForTag(fdef.Tag.Get("env"))
		if envValue != "" {
			switch f.Type().Kind() {
			case reflect.Int:
				if newValue, err := strconv.ParseInt(envValue, 10, 64); err == nil {
					t.Field(i).SetInt(newValue)
					updatedConfig = true
				} else {
					panic(err)
				}
			case reflect.Bool:
				if newValue, err := strconv.ParseBool(envValue); err == nil {
					t.Field(i).SetBool(newValue)
					updatedConfig = true
				} else {
					panic(err)
				}
			case reflect.String:
				t.Field(i).SetString(envValue)
				updatedConfig = true
			}
		}
	}
	return updatedConfig
}

// ParseConfig parses Relay's raw config and then produces a final
// version incorporating default values and environment variable values.
// Finalization proceeds using the following steps:
// 1. Parse YAML and populate new Config struct instance with values
// 2. Set default values on unassigned fields (for fields with defaults)
// 3. Apply environment variable overrides
// 4. Validate the finalized config
func ParseConfig(rawConfig []byte) (*Config, error) {
	config := Config{}
	err := yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing YAML: %s", err)
	}
	if config.Version != RelayConfigVersion {
		return nil, fmt.Errorf("Only Cog Relay config version %d is supported.", RelayConfigVersion)
	}
	applyDefaults(&config)
	applyEnvVars(&config)
	// Remove Docker config if it's disabled
	if config.DockerDisabled == true {
		config.Docker = nil
	}
	govalidator.TagMap["hostorip"] = govalidator.Validator(func(value string) bool {
		return govalidator.IsHost(value)
	})
	govalidator.TagMap["dockersocket"] = govalidator.Validator(func(value string) bool {
		return strings.HasPrefix(value, "unix://") || strings.HasPrefix(value, "tcp://")
	})
	_, err = govalidator.ValidateStruct(config)
	if err == nil && config.Docker != nil {
		_, err = govalidator.ValidateStruct(config.Docker)
		return &config, err
	}
	return &config, err
}

// LoadConfig reads a config file off disk and passes the contents to
// ParseConfig.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("Path to configuration file is required")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("Config file '%s' doesn't exist or is unreadable", path)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading config file '%s': %s",
			path, err)
	}
	return ParseConfig(buf)
}
