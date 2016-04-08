package engines

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/messages"
	"strings"
)

func BuildEnvironment(request messages.ExecutionRequest) []string {
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

	retval := make([]string, len(vars))
	i := 0
	for k, v := range vars {
		retval[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	log.Debugf("Calling environment: %v", retval)
	return retval
}
