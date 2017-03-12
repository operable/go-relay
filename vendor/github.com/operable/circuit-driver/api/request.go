package api

import (
	"bytes"
	"fmt"
	"os/exec"
)

func NewExecRequest() *ExecRequest {
	return &ExecRequest{
		Env: []*EnvVar{},
	}
}

func (er *ExecRequest) FindEnv(name string) string {
	for _, ev := range er.Env {
		if *(ev.Name) == name {
			return *(ev.Value)
		}
	}
	return ""
}

func (er *ExecRequest) PutEnv(name, value string) {
	ev := new(EnvVar)
	ev.Name = &name
	ev.Value = &value
	er.Env = append(er.Env, ev)
}

func (er *ExecRequest) DelEnv(name string) {
	for i, ev := range er.Env {
		if *(ev.Name) == name {
			er.Env = append(er.Env[:i], er.Env[i+1:]...)
			return
		}
	}
}

func (er *ExecRequest) SetExecutable(name string) {
	er.Executable = &name
}

// ToExecCommand builds a Go os/exec.Cmd from a source
// ExecRequest
func (er *ExecRequest) ToExecCommand() exec.Cmd {
	var command exec.Cmd
	command.Path = er.GetExecutable()
	command.Env = er.convertEnv()
	command.Stdin = bytes.NewBuffer(er.Stdin)
	return command
}

func (er *ExecRequest) convertEnv() []string {
	retval := []string{}
	for _, kv := range er.Env {
		retval = append(retval, fmt.Sprintf("%s=%v", kv.GetName(), kv.GetValue()))
	}
	return retval
}
