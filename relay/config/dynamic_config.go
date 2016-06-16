package config

import (
	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// LoadDynamicConfig loads the dyanmic configuration for a bundle if
// a) dynamic configuration is enabled and b) a config file exists for
// the requested bundle
func (c *Config) LoadDynamicConfig(bundle string) map[string]interface{} {
	retval := make(map[string]interface{})
	if c.DynamicConfigRoot == "" {
		log.Debugf("Dynamic config disabled.")
		return retval
	}
	if fullPath := locateConfigFile(c.DynamicConfigRoot, bundle); fullPath != "" {
		buf, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Errorf("Reading dynamic config for bundle %s failed: %s.", bundle, err)
			return retval
		}

		err = yaml.Unmarshal(buf, &retval)
		if err != nil {
			log.Errorf("Parsing dynamic config for bundle %s failed: %s.", bundle, err)
			return retval
		}
		for k := range retval {
			if strings.HasPrefix(k, "COG_") || strings.HasPrefix(k, "RELAY_") {
				delete(retval, k)
				log.Infof("Deleted illegal key %s from dynamic config for bundle %s.", k, bundle)
			}
		}
	}
	return retval
}

func locateConfigFile(configRoot string, bundle string) string {
	fullYamlPath := path.Join(configRoot, bundle, "config.yaml")
	fullYmlPath := path.Join(configRoot, bundle, "config.yml")
	yamlInfo, yamlErr := os.Stat(fullYamlPath)
	if yamlErr != nil || yamlInfo.IsDir() == true {
		ymlInfo, ymlErr := os.Stat(fullYmlPath)
		if ymlErr != nil || ymlInfo.IsDir() == true {
			log.Debugf("Dynamic config not found. Checked: '%s' and '%s'.",
				fullYamlPath, fullYmlPath)
			return ""
		}
		return fullYmlPath
	}
	return fullYamlPath
}
