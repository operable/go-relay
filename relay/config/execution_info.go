package config

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// ExecutionInfo applies to every container for a given Relay host
type ExecutionInfo struct {
	ExtraEnv       []string `yaml:"env" env:"RELAY_CONTAINER_ENV"`
	ParsedExtraEnv map[string]string
}

func (execution *ExecutionInfo) parse() {
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
