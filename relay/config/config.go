package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
	"github.com/go-yaml/yaml"
)

const (
	// RelayConfigVersion describes the compatible Relay config file format version
	RelayConfigVersion = 1
)

// Available execution engines
const (
	DockerEngine = "docker"
	NativeEngine = "native"
)

var validEngineNames = []string{DockerEngine, NativeEngine}

// Config is the top level struct for all Relay configuration
type Config struct {
	Version              int      `yaml:"version" valid:"int64,required"`
	ID                   string   `yaml:"id" env:"RELAY_ID" valid:"uuid,required"`
	MaxConcurrent        int      `yaml:"max_concurrent" env:"RELAY_MAX_CONCURRENT" valid:"int64,required" default:"16"`
	DynamicConfigRoot    string   `yaml:"dynamic_config_root" env:"RELAY_DYNAMIC_CONFIG_ROOT" valid:"-"`
	LogLevel             string   `yaml:"log_level" env:"RELAY_LOG_LEVEL" valid:"required" default:"info"`
	LogJSON              bool     `yaml:"log_json" env:"RELAY_LOG_JSON" valid:"bool" default:"false"`
	LogPath              string   `yaml:"log_path" env:"RELAY_LOG_PATH" valid:"required" default:"stdout"`
	Cog                  *CogInfo `yaml:"cog" valid:"required"`
	EnginesEnabled       string   `yaml:"enabled_engines" env:"RELAY_ENABLED_ENGINES" valid:"exec_engines" default:"docker,native"`
	ParsedEnginesEnabled []string
	Docker               *DockerInfo    `yaml:"docker" valid:"-"`
	Execution            *ExecutionInfo `yaml:"execution" valid:"-"`
}

// CogInfo contains information required to connect to an upstream Cog host
type CogInfo struct {
	Host            string `yaml:"host" env:"RELAY_COG_HOST" valid:"hostorip,required" default:"127.0.0.1"`
	Port            int    `yaml:"port" env:"RELAY_COG_PORT" valid:"int64,required" default:"1883"`
	Token           string `yaml:"token" env:"RELAY_COG_TOKEN" valid:"required"`
	SSLEnabled      bool   `yaml:"enable_ssl" env:"RELAY_COG_ENABLE_SSL" valid:"bool" default:"false"`
	RefreshInterval string `yaml:"refresh_interval" env:"RELAY_COG_REFRESH_INTERVAL" valid:"required" default:"15m"`
}

// DockerInfo contains information required to interact with dockerd and external Docker registries
type DockerInfo struct {
	UseEnv           bool   `yaml:"use_env" env:"RELAY_DOCKER_USE_ENV" valid:"-" default:"false"`
	SocketPath       string `yaml:"socket_path" env:"RELAY_DOCKER_SOCKET_PATH" valid:"dockersocket,required" default:"unix:///var/run/docker.sock"`
	CleanInterval    string `yaml:"clean_interval" env:"RELAY_DOCKER_CLEAN_INTERVAL" valid:"required" default:"5m"`
	RegistryHost     string `yaml:"registry_host" env:"RELAY_DOCKER_REGISTRY_HOST" valid:"host,required" default:"index.docker.io"`
	RegistryUser     string `yaml:"registry_user" env:"RELAY_DOCKER_REGISTRY_USER" valid:"-"`
	RegistryEmail    string `yaml:"registry_email" env:"RELAY_DOCKER_REGISTRY_EMAIL" valid:"-"`
	RegistryPassword string `yaml:"registry_password" env:"RELAY_DOCKER_REGISTRY_PASSWORD" valid:"-"`
}

// ExecutionInfo applies to every container for a given Relay host
type ExecutionInfo struct {
	ExtraEnv       []string `yaml:"env" env:"RELAY_CONTAINER_ENV"`
	ParsedExtraEnv map[string]string
}

// RefreshDuration returns RefreshInterval as a time.Duration
func (c *Config) RefreshDuration() time.Duration {
	duration, err := time.ParseDuration(c.Cog.RefreshInterval)
	if err != nil {
		panic(fmt.Errorf("Error parsing refresh_interval: %s", err))
	}
	return duration
}

// DockerEnabled returns true when enabled_engines includes "docker"
func (c *Config) DockerEnabled() bool {
	return c.engineEnabled(DockerEngine)
}

// NativeEnabled returns true when enabled_engines includes "native"
func (c *Config) NativeEnabled() bool {
	return c.engineEnabled(NativeEngine)
}

func (c *Config) engineEnabled(name string) bool {
	for _, v := range c.ParsedEnginesEnabled {
		if v == name {
			return true
		}
	}
	return false
}

// URL returns a MQTT URL for the upstream Cog host
func (ci *CogInfo) URL() string {
	proto := "tcp"
	if ci.SSLEnabled {
		proto = "ssl"
	}
	return fmt.Sprintf("%s://%s:%d", proto, ci.Host, ci.Port)
}

// CleanDuration returns CleanInterval as a time.Duration
func (di *DockerInfo) CleanDuration() time.Duration {
	duration, err := time.ParseDuration(di.CleanInterval)
	if err != nil {
		panic(fmt.Errorf("Error parsing docker/clean_interval: %s", err))
	}
	return duration
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

func parseExtraEnv(execution *ExecutionInfo) {
	execution.ParsedExtraEnv = make(map[string]string)
	for _, v := range execution.ExtraEnv {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			panic(fmt.Errorf("Illegal environment var specification in execution/env: %s", v))
		}
		if strings.HasPrefix(parts[0], "COG_") || strings.HasPrefix(parts[0], "RELAY_") {
			log.Infof("Deleted illegal key %s from exection/env.", parts[0])
		} else {
			execution.ParsedExtraEnv[parts[0]] = parts[1]
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

func isValidEngineName(name string) bool {
	for _, v := range validEngineNames {
		if name == v {
			return true
		}
	}
	return false
}

func parseEngines(relayConfig *Config) {
	engines := strings.Split(relayConfig.EnginesEnabled, ",")
	parsed := []string{}
	for _, engine := range engines {
		engine = strings.Trim(engine, " ")
		if isValidEngineName(engine) {
			dupe := false
			for _, p := range parsed {
				// engine is a duplicate
				if p == engine {
					dupe = true
					break
				}
			}
			if dupe == false {
				parsed = append(parsed, engine)
			}
		}
	}
	relayConfig.ParsedEnginesEnabled = parsed
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
	parseEngines(&config)
	// Remove Docker config if it's disabled
	if config.DockerEnabled() == false {
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
	if config.Execution == nil {
		config.Execution = &ExecutionInfo{
			ExtraEnv:       []string{},
			ParsedExtraEnv: make(map[string]string),
		}
	}
	parseExtraEnv(config.Execution)
	return &config, err
}

// LoadConfig reads a config file off disk and passes the contents to
// ParseConfig.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("Path to configuration file is required.")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("Config file '%s' doesn't exist or is unreadable.", path)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading config file '%s': %s.",
			path, err)
	}
	return ParseConfig(buf)
}
