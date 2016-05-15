package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

var errorMissingConfigPath = errors.New("Path to configuration file is required.")
var errorConfigFileNotFound = errors.New("Config file not found or is unreadable.")
var errorWrongConfigVersion = fmt.Errorf("Only Cog Relay config version %d is supported.", RelayConfigVersion)

// LoadConfig reads a config file off disk
func LoadConfig(path string) (RawConfig, error) {
	if path == "" {
		return nil, errorMissingConfigPath
	}
	if _, err := os.Stat(path); err != nil {
		return nil, errorConfigFileNotFound
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rawConfig := RawConfig(buf)
	return rawConfig, nil
}
