package messages

import (
	"time"
)

type ExecCommandRequest struct {
	Executable string
	WorkingDir string
	CogEnv     []byte
	Env        []string
	Die        bool
}

type ExecCommandResponse struct {
	Executable string
	Success    bool
	Stdout     []byte
	Stderr     []byte
	Elapsed    time.Duration
	Dead       bool
}
