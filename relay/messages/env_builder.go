package messages

import (
	"fmt"
	"github.com/operable/circuit-driver/api"
	"github.com/operable/go-relay/relay/config"
	"strings"
)

func (er *ExecutionRequest) compileEnvironment(request *api.ExecRequest, relayConfig *config.Config, useDynamicConfig bool) bool {
	for i, v := range er.Args {
		request.PutEnv(fmt.Sprintf("COG_ARGV_%d", i), fmt.Sprintf("%v", v))
	}
	request.PutEnv("COG_ARGC", fmt.Sprintf("%d", len(er.Args)))
	if len(er.Options) > 0 {
		cogOpts := ""
		for k, v := range er.Options {
			optName := fmt.Sprintf("COG_OPT_%s", strings.ToUpper(k))
			request.PutEnv(optName, fmt.Sprintf("%v", v))
			if cogOpts == "" {
				cogOpts = k
			} else {
				cogOpts = fmt.Sprintf("%s,%s", cogOpts, k)
			}
		}
		request.PutEnv("COG_OPTS", cogOpts)
	}
	request.PutEnv("COG_BUNDLE", er.BundleName())
	request.PutEnv("COG_COMMAND", er.CommandName())
	request.PutEnv("COG_CHAT_HANDLE", er.Requestor.Handle)
	request.PutEnv("COG_PIPELINE_ID", er.PipelineID())
	request.PutEnv("COG_SERVICE_TOKEN", er.ServiceToken)
	request.PutEnv("COG_SERVICES_ROOT", er.ServicesRoot)
	request.PutEnv("COG_INVOCATION_ID", er.InvocationID)

	if er.InvocationStep != "" {
		request.PutEnv("COG_INVOCATION_STEP", er.InvocationStep)
	}

	foundDynamicConfig := false
	if useDynamicConfig {
		dyn := relayConfig.LoadDynamicConfig(er.BundleName(), er.Room.Name, er.User.Username)
		foundDynamicConfig = len(dyn) > 0
		for k, v := range dyn {
			request.PutEnv(k, fmt.Sprintf("%s", v))
		}
	}

	if relayConfig.Execution != nil {
		for k, v := range relayConfig.Execution.ParsedExtraEnv {
			request.PutEnv(k, v)
		}
	}

	return foundDynamicConfig
}
