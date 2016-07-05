package messages

import (
	"fmt"
	"github.com/operable/circuit-driver/api"
	"github.com/operable/go-relay/relay/config"
	"os"
	"strings"
)

func (psr *PipelineStageRequest) buildBaseVars() {
	psr.vars = make(map[string]string)
	psr.vars["PATH"] = "/bin:/usr/bin"
	psr.vars["COG_BUNDLE"] = psr.BundleName()
	psr.vars["COG_COMMAND"] = psr.commandName
	psr.vars["COG_CHAT_HANDLE"] = psr.Requestor.Handle
	psr.vars["COG_PIPELINE_ID"] = psr.PipelineID()
	psr.vars["COG_SERVICE_TOKEN"] = psr.ServiceToken
	psr.vars["COG_SERVICES_ROOT"] = psr.ServicesRoot
	psr.vars["COG_INVOCATION_ID"] = psr.InvocationID
	psr.vars["USER"] = os.Getenv("USER")
	psr.vars["HOME"] = os.Getenv("HOME")
	psr.vars["LANG"] = os.Getenv("LANG")
}

func (psr *PipelineStageRequest) compileEnvironment(request *api.ExecRequest, er ExecutionRequest, relayConfig *config.Config,
	useDynamicConfig bool) bool {
	if psr.vars == nil {
		psr.buildBaseVars()
	}
	for k, v := range psr.vars {
		request.PutEnv(k, v)
	}
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

	if er.InvocationStep != "" {
		request.PutEnv("COG_INVOCATION_STEP", er.InvocationStep)
	}

	foundDynamicConfig := false
	if useDynamicConfig {
		dyn := relayConfig.LoadDynamicConfig(psr.BundleName())
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
