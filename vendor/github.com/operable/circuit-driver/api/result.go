package api

import (
	"time"
)

func (er *ExecResult) SetElapsed(elapsed time.Duration) {
	e := int64(elapsed * time.Nanosecond)
	er.Elapsed = &e
}

func (er *ExecResult) SetSuccess(flag bool) {
	er.Success = &flag
}
