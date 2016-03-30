package relay

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
	CONFIG_VERSION = 1
)

// Top level configuration struct
type Config struct {
	Version       int            `yaml:"version" valid:"int64,required"`
	ID            string         `yaml:"id" env:"RELAY_ID" valid:"uuid,required"`
	MaxConcurrent int            `yaml:"max_concurrent" env:"RELAY_MAX_CONCURRENT" valid:"int64,required" default:"16"`
	LogLevel      string         `yaml:"log_level" env:"RELAY_LOG_LEVEL" valid:"required" default:"info"`
	LogJSON       bool           `yaml:"log_json" env:"RELAY_LOG_JSON" default:"false"`
	LogPath       string         `yaml:"log_path" env:"RELAY_LOG_PATH" valid:"required" default:"stdout"`
	Cog           *CogInfo       `yaml:"cog" valid:"required"`
	Docker        *DockerInfo    `yaml:"docker" valid:"-"`
	Execution     *ExecutionInfo `yaml:"execution" valid:"required"`
}

// Information about upstream Cog instance
type CogInfo struct {
	ID    string `yaml:"id" env:"RELAY_COG_ID" valid:"uuid,required"`
	Host  string `yaml:"host" env:"RELAY_COG_HOST" valid:"hostorip,required" default:"127.0.0.1"`
	Port  int    `yaml:"port" env:"RELAY_COG_PORT" valid:"int64,required" default:"1883"`
	Token string `yaml:"token" env:"RELAY_COG_TOKEN" valid:"required"`
}

// Information about Docker install
type DockerInfo struct {
	UseEnv           bool   `yaml:"use_env" env:"RELAY_DOCKER_USE_ENV" valid:"-" default:"false"`
	SocketPath       string `yaml:"socket_path" env:"RELAY_DOCKER_SOCKET_PATH" valid:"dockersocket,required" default:"unix:///var/run/docker.sock"`
	RegistryHost     string `yaml:"registry_host" env:"RELAY_DOCKER_REGISTRY_HOST" valid:"host,required" default:"hub.docker.com"`
	RegistryUser     string `yaml:"registry_user" env:"RELAY_DOCKER_REGISTRY_USER" valid:"printableascii,required"`
	RegistryPassword string `yaml:"registry_password" env:"RELAY_DOCKER_REGISTRY_PASSWORD" valid:"required"`
}

// Configuration parameters applied to every container
// started by a given Relay
type ExecutionInfo struct {
	CPUShares int64    `yaml:"cpu_shares" env:"RELAY_CONTAINER_CPUSHARES" valid:"int64"`
	CPUSet    string   `yaml:"cpu_set" env:"RELAY_CONTAINER_CPUSET"`
	ExtraEnv  []string `yaml:"env" env:"RELAY_CONTAINER_ENV"`
}

func applyDefaults(config *Config) {
	setDefaultValues(config)
	if config.Cog != nil {
		setDefaultValues(config.Cog)
	}
	if config.Docker != nil {
		setDefaultValues(config.Docker)
	}
	if config.Execution != nil {
		setDefaultValues(config.Execution)
	}
}

func setDefaultValues(config interface{}) {
	t := reflect.ValueOf(config).Elem()
	configType := t.Type()
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
					} else {
						panic(err)
					}
				}
			case reflect.Bool:
				if newValue, err := strconv.ParseBool(defaultValue); err == nil {
					if currentValue == false && newValue == true {
						t.Field(i).SetBool(newValue)
					}
				} else {
					panic(err)
				}
			case reflect.String:
				if currentValue == "" {
					t.Field(i).SetString(defaultValue)
				}
			}
		}
	}
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

func setEnvVars(config interface{}) {
	t := reflect.ValueOf(config).Elem()
	configType := t.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fdef := configType.Field(i)
		envValue := envVarForTag(fdef.Tag.Get("env"))
		if envValue != "" {
			switch f.Type().Kind() {
			case reflect.Int:
				if newValue, err := strconv.ParseInt(envValue, 10, 64); err == nil {
					t.Field(i).SetInt(newValue)
				} else {
					panic(err)
				}
			case reflect.Bool:
				if newValue, err := strconv.ParseBool(envValue); err == nil {
					t.Field(i).SetBool(newValue)
				} else {
					panic(err)
				}
			case reflect.String:
				t.Field(i).SetString(envValue)
			}
		}
	}
}

// Parses Relay's config file. Finalizing Relay's configuration uses the following steps:
// 1. Parse YAML and populate new Config struct instance with values
// 2. Set default values on unassigned fields (for fields with defaults)
// 3. Apply environment variable overrides
// 4. Validate the finalized config
func ParseConfig(rawConfig []byte) (*Config, error) {
	config := Config{}
	err := yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing YAML: %s", err))
	}
	if config.Version != CONFIG_VERSION {
		return nil, errors.New(fmt.Sprintf("Only Cog Relay config version %d is supported.", CONFIG_VERSION))
	}
	applyDefaults(&config)
	applyEnvVars(&config)
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

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("Path to configuration file is required")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, errors.New(fmt.Sprintf("Config file '%s' doesn't exist or is unreadable", path))
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading config file '%s': %s",
			path, err))
	}
	return ParseConfig(buf)
}
