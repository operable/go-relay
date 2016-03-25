package relay

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/go-yaml/yaml"
)

// Top level configuration struct
type Config struct {
	ID            string         `yaml:"id" env:"RELAY_ID" valid:"uuid"`
	MaxConcurrent int            `yaml:"max_concurrent" env:"RELAY_MAX_CONCURRENT" valid:"-" default:"16"`
	Cog           *CogInfo       `yaml:"cog"`
	Docker        *DockerInfo    `yaml:"docker"`
	Execution     *ExecutionInfo `yaml:"execution"`
}

// Information about upstream Cog instance
type CogInfo struct {
	ID   string `yaml:"id" env:"RELAY_COG_ID" valid:"uuid"`
	Host string `yaml:"host" env:"RELAY_COG_HOST" valid:"host_or_ip" default:"127.0.0.1"`
	Port int    `yaml:"port" env:"RELAY_COG_PORT" valid:"-" default:"1883"`
}

// Information about Docker install
type DockerInfo struct {
	UseEnv     bool   `yaml:"use_env" valid:"-" default:"false"`
	SocketPath string `yaml:"socket_path" env:"RELAY_DOCKER_SOCKET_PATH" valid:"-" default:"/var/run/docker.sock"`
}

// Configuration parameters applied to every container
// started by a given Relay
type ExecutionInfo struct {
	CPUShares int64    `yaml:"cpu_shares" env:"RELAY_CONTAINER_CPUSHARES"`
	CPUSet    string   `yaml:"cpu_set" env:"RELAY_CONTAINER_CPUSET"`
	ExtraEnv  []string `yaml:"env" env:"RELAY_CONTAINER_ENV"`
}

func applyDefaults(config *Config) {
	setDefaultValues(config)
	setDefaultValues(config.Cog)
	setDefaultValues(config.Docker)
	setDefaultValues(config.Execution)
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
	setEnvVars(config.Cog)
	setEnvVars(config.Docker)
	setEnvVars(config.Execution)
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
	applyDefaults(&config)
	applyEnvVars(&config)
	govalidator.TagMap["host_or_ip"] = govalidator.Validator(func(value string) bool {
		return govalidator.IsIP(value) || govalidator.IsHost(value)
	})
	_, err = govalidator.ValidateStruct(config)
	if err != nil {
		return nil, err
	}
	return &config, nil
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
