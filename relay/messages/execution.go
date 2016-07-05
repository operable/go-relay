package messages

import (
	"encoding/json"
	"github.com/operable/circuit-driver/api"
	"github.com/operable/go-relay/relay/config"
	"strings"
)

// ExecutionRequest is a request to execute a command
// as part of a Cog pipeline
type ExecutionRequest struct {
	Options        map[string]interface{} `json:"options"`
	Args           []interface{}          `json:"args"`
	CogEnv         interface{}            `json:"cog_env"`
	InvocationStep string                 `json:"invocation_step"`
}

type PipelineStageRequest struct {
	InvocationID  string                 `json:"invocation_id"`
	Requests      []ExecutionRequest     `json:"requests"`
	ReplyTo       string                 `json:"reply_to"`
	Requestor     ChatUser               `json:"requestor"`
	User          CogUser                `json:"user"`
	ServiceToken  string                 `json:"service_token"`
	ServicesRoot  string                 `json:"services_root"`
	Room          string                 `json:"room"`
	Command       string                 `json:"command"`
	CommandConfig map[string]interface{} `json:"command_config"`
	bundleName    string
	commandName   string
	pipelineID    string
	vars          map[string]string
}

func (psr *PipelineStageRequest) Parse() {
	commandParts := strings.SplitN(psr.Command, ":", 2)
	pipelineParts := strings.SplitN(psr.ReplyTo, "/", 5)
	psr.bundleName = commandParts[0]
	psr.commandName = commandParts[1]
	psr.pipelineID = pipelineParts[3]
}

func (psr *PipelineStageRequest) BundleName() string {
	return psr.bundleName
}

func (psr *PipelineStageRequest) PipelineID() string {
	return psr.pipelineID
}

// ChatUser contains chat information about the submittor
type ChatUser struct {
	ID       interface{} `json:"id"` // Slack IDs are strings, HipChat are integers
	Handle   string      `json:"handle"`
	Provider string      `json:"provider"`
}

// CogUser contains Cog user information about the submittor
type CogUser struct {
	ID        string `json:"id"`
	Email     string `json:"email_address"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// ExecutionResponse contains the results of executing a command
type ExecutionResponse struct {
	Room          string      `json:"room"`
	Bundle        string      `json:"bundle"`
	Status        string      `json:"status"`
	StatusMessage string      `json:"status_message"`
	Template      string      `json:"template,omitempty"`
	Body          interface{} `json:"body"`
	IsJSON        bool        `json:"omit"`
}

type PipelineStageResponse struct {
	Responses []ExecutionResponse `json:"responses"`
}

// ToCircuitRequest converts an ExecutionRequest into a circuit.api.ExecRequest
func (psr *PipelineStageRequest) ToCircuitRequest(er *ExecutionRequest, bundle *config.Bundle,
	relayConfig *config.Config, useDynamicConfig bool) (*api.ExecRequest, bool) {
	retval := &api.ExecRequest{}
	hasDynamicConfig := psr.compileEnvironment(retval, *er, relayConfig, useDynamicConfig)
	retval.SetExecutable(bundle.Commands[psr.commandName].Executable)
	if er.CogEnv != nil {
		jenv, _ := json.Marshal(er.CogEnv)
		retval.Stdin = jenv
	}
	return retval, hasDynamicConfig
}
