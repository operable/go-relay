package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
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
var errorNoExecutionEngines = errors.New("Invalid Relay configuration detected. At least one execution engine must be enabled.")
var errorMissingDynamicConfigRoot = errors.New("Enabling 'managed_dynamic_config' requires setting 'dynamic_config_root'.")
var errorBadDynConfigInterval = errors.New("Error parsing managed_dynamic_config_interval")

// Config is the top level struct for all Relay configuration
type Config struct {
	Version               int      `yaml:"version" valid:"int64,required"`
	ID                    string   `yaml:"id" env:"RELAY_ID" valid:"uuid,required"`
	MaxConcurrent         int      `yaml:"max_concurrent" env:"RELAY_MAX_CONCURRENT" valid:"int64,required" default:"16"`
	DynamicConfigRoot     string   `yaml:"dynamic_config_root" env:"RELAY_DYNAMIC_CONFIG_ROOT" valid:"-"`
	ManagedDynamicConfig  bool     `yaml:"managed_dynamic_config" env:"RELAY_MANAGED_DYNAMIC_CONFIG" valid:"-"`
	DynamicConfigInterval string   `yaml:"managed_dynamic_config_interval" env:"RELAY_MANAGED_DYNAMIC_CONFIG_INTERVAL" default:"5s"`
	LogLevel              string   `yaml:"log_level" env:"RELAY_LOG_LEVEL" valid:"required" default:"info"`
	LogJSON               bool     `yaml:"log_json" env:"RELAY_LOG_JSON" valid:"bool" default:"false"`
	LogPath               string   `yaml:"log_path" env:"RELAY_LOG_PATH" valid:"required" default:"stdout"`
	Cog                   *CogInfo `yaml:"cog" valid:"required"`
	EnginesEnabled        string   `yaml:"enabled_engines" env:"RELAY_ENABLED_ENGINES" valid:"exec_engines" default:"docker,native"`
	ParsedEnginesEnabled  []string
	Docker                *DockerInfo    `yaml:"docker" valid:"-"`
	Execution             *ExecutionInfo `yaml:"execution" valid:"-"`
}

// RefreshDuration returns RefreshInterval as a time.Duration
func (c *Config) RefreshDuration() time.Duration {
	duration, err := time.ParseDuration(c.Cog.RefreshInterval)
	if err != nil {
		panic(fmt.Errorf("Error parsing refresh_interval: %s", err))
	}
	return duration
}

// ManagedDynamicConfigRefreshDuration returns DynamicConfigInterval as a time.Duration
func (c *Config) ManagedDynamicConfigRefreshDuration() time.Duration {
	duration, err := time.ParseDuration(c.DynamicConfigInterval)
	if err != nil {
		panic(errorBadDynConfigInterval)
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

// Verify sanity checks the configuration to ensure it's correct
func (c *Config) Verify() error {
	if c.DockerEnabled() == false && c.NativeEnabled() == false {
		return errorNoExecutionEngines
	}
	if c.ManagedDynamicConfig == true && c.DynamicConfigRoot == "" {
		return errorMissingDynamicConfigRoot
	}
	if c.ManagedDynamicConfig == true {
		c.DynamicConfigRoot = path.Join(c.DynamicConfigRoot, "managed")
	}
	return nil
}

func (c *Config) populate() {
	setDefaultValues(c)
	setEnvVars(c)
	if c.Cog == nil {
		c.Cog = &CogInfo{}
	}
	setDefaultValues(c.Cog)
	setEnvVars(c.Cog)
	if c.Docker == nil {
		c.Docker = &DockerInfo{}
	}
	setDefaultValues(c.Docker)
	setEnvVars(c.Docker)
	if c.Execution == nil {
		c.Execution = &ExecutionInfo{}
	}
	setDefaultValues(c.Execution)
	setEnvVars(c.Execution)
	c.Execution.parse()
	c.parseEngines()
}

func (c *Config) parseEngines() {
	engines := strings.Split(c.EnginesEnabled, ",")
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
	c.ParsedEnginesEnabled = parsed
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
