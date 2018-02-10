package config

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// LoadDynamicConfig loads the dyanmic configuration for a bundle if
// a) dynamic configuration is enabled and b) a config file exists for
// the requested bundle. Room- and user-specific configurations are
// layered on top, if they exist.
//
// If any configuration files exist, but cannot be properly processed
// (read, parsed as YAML, etc), an empty map is returned.
func (c *Config) LoadDynamicConfig(bundle string, roomName string, userName string) map[string]interface{} {
	retval := make(map[string]interface{})
	if c.DynamicConfigRoot == "" {
		log.Debugf("Dynamic config disabled.")
		return retval
	}

	rootInfo, rootErr := os.Stat(c.DynamicConfigRoot)
	if rootErr != nil || rootInfo.IsDir() == false {
		log.Debugf("Dynamic configuration root dir not found.")
		return retval
	}

	configs := allConfigFiles(c.DynamicConfigRoot, bundle, roomName, userName)
	for _, f := range configs {
		log.Debugf("Reading configuration file: %s", f)
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			log.Errorf("Reading dynamic config file %s failed: %s.", f, err)
			return make(map[string]interface{})
		}

		// by unmarshalling into the same map for each layer, we
		// achieve a shallow merge of the layers, with successive
		// top-level keys overwriting values previously set.
		err = yaml.Unmarshal(buf, &retval)
		if err != nil {
			log.Errorf("Parsing dynamic config file %s failed: %s.", f, err)
			return make(map[string]interface{})
		}
	}

	for k := range retval {
		if strings.HasPrefix(k, "COG_") || strings.HasPrefix(k, "RELAY_") {
			delete(retval, k)
			log.Infof("Deleted illegal key %s from dynamic config for bundle %s.", k, bundle)
		}
	}

	return retval
}

// Given the config root, a bundle name, a Cog username, and a chat
// room name, return a list of all the filenames to be consolidated,
// in the order they should be layered.
func allConfigFiles(configRoot string, bundle string, room string, user string) []string {
	var configs []string

	if path := resolveYAMLFile(path.Join(configRoot, bundle, "config")); path != "" {
		configs = append(configs, path)
	}

	roomFile := fmt.Sprintf("room_%s", strings.ToLower(room))
	if path := resolveYAMLFile(path.Join(configRoot, bundle, roomFile)); path != "" {
		configs = append(configs, path)
	}

	userFile := fmt.Sprintf("user_%s", strings.ToLower(user))
	if path := resolveYAMLFile(path.Join(configRoot, bundle, userFile)); path != "" {
		configs = append(configs, path)
	}

	return configs
}

// Given a base file name, return the path to either the yaml or yml
// version, returning the ".yaml" version preferentially.
func resolveYAMLFile(base string) string {

	fullYamlPath := fmt.Sprint(base, ".yaml")

	yamlInfo, yamlErr := os.Stat(fullYamlPath)
	if yamlErr != nil || yamlInfo.IsDir() == true {
		fullYmlPath := fmt.Sprint(base, ".yml")
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
