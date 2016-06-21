package messages

import (
	"fmt"
	"github.com/operable/go-relay/relay/config"
	"os"
	"strings"
)

func (er *ExecutionRequest) compileEnvironment(relayConfig *config.Config, useDynamicConfig bool) (map[string]string, bool) {
	vars := make(map[string]string)
	vars["PATH"] = "/bin:/usr/bin"
	for i, v := range er.Args {
		argName := fmt.Sprintf("COG_ARGV_%d", i)
		vars[argName] = fmt.Sprintf("%v", v)
	}
	vars["COG_ARGC"] = fmt.Sprintf("%d", len(er.Args))
	if len(er.Options) > 0 {
		cogOpts := ""
		for k, v := range er.Options {
			optName := fmt.Sprintf("COG_OPT_%s", strings.ToUpper(k))
			vars[optName] = fmt.Sprintf("%v", v)
			if cogOpts == "" {
				cogOpts = k
			} else {
				cogOpts = fmt.Sprintf("%s,%s", cogOpts, k)
			}
		}
		vars["COG_OPTS"] = cogOpts
	}
	vars["COG_BUNDLE"] = er.BundleName()
	vars["COG_COMMAND"] = er.CommandName()
	vars["COG_CHAT_HANDLE"] = er.Requestor.Handle
	vars["COG_PIPELINE_ID"] = er.PipelineID()
	vars["COG_SERVICE_TOKEN"] = er.ServiceToken
	vars["COG_SERVICES_ROOT"] = er.ServicesRoot
	vars["COG_INVOCATION_ID"] = er.InvocationID

	if er.InvocationStep != "" {
		vars["COG_INVOCATION_STEP"] = er.InvocationStep
	}

	foundDynamicConfig := false
	if useDynamicConfig {
		dyn := relayConfig.LoadDynamicConfig(er.BundleName())
		foundDynamicConfig = len(dyn) > 0
		for k, v := range dyn {
			vars[k] = fmt.Sprintf("%s", v)
		}
	}

	if relayConfig.Execution != nil {
		for k, v := range relayConfig.Execution.ParsedExtraEnv {
			vars[k] = v
		}
	}

	vars["USER"] = os.Getenv("USER")
	vars["HOME"] = os.Getenv("HOME")
	vars["LANG"] = os.Getenv("LANG")
	return vars, foundDynamicConfig
}
