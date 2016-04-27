package engines

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// BuildEnvironment constructs the calling environment for a command
func BuildEnvironment(request messages.ExecutionRequest, relayConfig config.Config) []string {
	vars := make(map[string]string)
	for i, v := range request.Args {
		argName := fmt.Sprintf("COG_ARGV_%d", i)
		vars[argName] = fmt.Sprintf("%s", v)
	}
	vars["COG_ARGC"] = fmt.Sprintf("%d", len(request.Args))
	if len(request.Options) > 0 {
		cogOpts := ""
		for k, v := range request.Options {
			optName := fmt.Sprintf("COG_OPT_%s", strings.ToUpper(k))
			vars[optName] = fmt.Sprintf("%s", v)
			if cogOpts == "" {
				cogOpts = k
			} else {
				cogOpts = fmt.Sprintf("%s,%s", cogOpts, k)
			}
		}
		vars["COG_OPTS"] = cogOpts
	}
	vars["COG_BUNDLE"] = request.BundleName()
	vars["COG_COMMAND"] = request.CommandName()
	vars["COG_CHAT_HANDLE"] = request.Requestor.Handle
	vars["COG_PIPELINE_ID"] = request.PipelineID()
	vars["COG_SERVICE_TOKEN"] = request.ServiceToken

	dyn := loadDynamicConfig(relayConfig, request.BundleName())
	if dyn != nil {
		for k, v := range *dyn {
			vars[k] = fmt.Sprintf("%s", v)
		}
	}

	if relayConfig.Execution != nil {
		for k, v := range relayConfig.Execution.ParsedExtraEnv {
			vars[k] = v
		}
	}

	retval := make([]string, len(vars))
	i := 0
	for k, v := range vars {
		retval[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	return retval
}

func loadDynamicConfig(relayConfig config.Config, bundle string) *map[string]interface{} {
	if relayConfig.DynamicConfigRoot == "" {
		log.Debugf("Dynamic config disabled.")
		return nil
	}
	if fullPath := locateConfigFile(relayConfig.DynamicConfigRoot, bundle); fullPath != "" {
		buf, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Errorf("Reading dynamic config for bundle %s failed: %s.", bundle, err)
			return nil
		}
		retval := make(map[string]interface{})
		err = yaml.Unmarshal(buf, &retval)
		if err != nil {
			log.Errorf("Parsing dynamic config for bundle %s failed: %s.", bundle, err)
			return nil
		}
		for k := range retval {
			if strings.HasPrefix(k, "COG_") || strings.HasPrefix(k, "RELAY_") {
				delete(retval, k)
				log.Infof("Deleted illegal key %s from dynamic config for bundle %s.", k, bundle)
			}
		}
		if len(retval) == 0 {
			return nil
		}
		return &retval
	}
	return nil
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
