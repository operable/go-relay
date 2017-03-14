package api

import (
	"bytes"
	"regexp"
	"time"
)

// Driver is the command execution interface
type Driver interface {
	Run(*ExecRequest) (ExecResult, error)
}

// BlockingDriver executes requests one-at-a-time
type BlockingDriver struct{}

var forkExecPrefix = regexp.MustCompile("^fork/exec ")

func (bd BlockingDriver) Run(request *ExecRequest) (ExecResult, error) {
	command := request.ToExecCommand()
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	start := time.Now()
	err := command.Run()
	finish := time.Now()
	result := ExecResult{}
	result.SetElapsed(finish.Sub(start))
	if err != nil {
		stderr.WriteString(forkExecPrefix.ReplaceAllString(err.Error(), ""))
		result.SetSuccess(false)
	} else {
		result.SetSuccess(true)
	}
	result.Stderr = stderr.Bytes()
	result.Stdout = stdout.Bytes()
	return result, nil
}
