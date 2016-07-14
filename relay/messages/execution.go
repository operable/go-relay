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
	InvocationID   string                 `json:"invocation_id"`
	InvocationStep string                 `json:"invocation_step"`
	Command        string                 `json:"command"`
	CommandConfig  map[string]interface{} `json:"command_config"`
	ReplyTo        string                 `json:"reply_to"`
	Requestor      ChatUser               `json:"requestor"`
	User           CogUser                `json:"user"`
	ServiceToken   string                 `json:"service_token"`
	ServicesRoot   string                 `json:"services_root"`
	bundleName     string
	commandName    string
	pipelineID     string
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
	Room           string      `json:"room"`
	Bundle         string      `json:"bundle"`
	Status         string      `json:"status"`
	StatusMessage  string      `json:"status_message"`
	Template       string      `json:"template,omitempty"`
	FormatPipeline string      `json:"format_pipeline,omitempty"`
	Body           interface{} `json:"body"`
	IsJSON         bool        `json:"omit"`
	Aborted        bool        `json:"omit"`
}

// ToCircuitRequest converts an ExecutionRequest into a circuit.api.ExecRequest
func (er *ExecutionRequest) ToCircuitRequest(bundle *config.Bundle, relayConfig *config.Config, useDynamicConfig bool) (*api.ExecRequest, bool) {
	retval := &api.ExecRequest{}
	hasDynamicConfig := er.compileEnvironment(retval, relayConfig, useDynamicConfig)
	executable := bundle.Commands[er.commandName].Executable
	retval.SetExecutable(executable)
	if er.CogEnv != nil {
		jenv, _ := json.Marshal(er.CogEnv)
		retval.Stdin = jenv
	}
	return retval, hasDynamicConfig
}

// BundleName returns just the bundle part of the
// command's fully qualified name
func (er *ExecutionRequest) BundleName() string {
	return er.bundleName
}

// CommandName returns just the command part of the
// command's fully qualified name
func (er *ExecutionRequest) CommandName() string {
	return er.commandName
}

// PipelineID returns the pipeline id assigned to
// this request
func (er *ExecutionRequest) PipelineID() string {
	return er.pipelineID
}

// Parse extracts bundle name, command name, and
// pipeline id
func (er *ExecutionRequest) Parse() {
	commandParts := strings.SplitN(er.Command, ":", 2)
	pipelineParts := strings.SplitN(er.ReplyTo, "/", 5)
	er.bundleName = commandParts[0]
	er.commandName = commandParts[1]
	er.pipelineID = pipelineParts[3]
}
