package circuit

import (
	"github.com/operable/circuit-driver/api"
)

type nativeEnvironment struct {
	options  CreateEnvironmentOptions
	requests chan api.ExecRequest
	results  chan api.ExecResult
	control  chan byte
	userData EnvironmentUserData
	driver   api.Driver
	isDead   bool
}

func (ne *nativeEnvironment) init(options CreateEnvironmentOptions) error {
	ne.options = options
	ne.driver = api.BlockingDriver{}
	ne.requests = make(chan api.ExecRequest)
	ne.results = make(chan api.ExecResult)
	ne.control = make(chan byte)
	go func() {
		ne.runWorker()
	}()
	return nil
}

func (ne *nativeEnvironment) runWorker() {
	for {
		select {
		case <-ne.control:
			break
		case request := <-ne.requests:
			result, _ := ne.driver.Run(&request)
			ne.results <- result
		}
	}
}

func (ne *nativeEnvironment) GetKind() EnvironmentKind {
	return NativeKind
}

func (ne *nativeEnvironment) SetUserData(data EnvironmentUserData) error {
	if ne.isDead {
		return ErrorDeadEnvironment
	}
	ne.userData = data
	return nil
}

func (ne *nativeEnvironment) GetUserData() (EnvironmentUserData, error) {
	if ne.isDead {
		return nil, ErrorDeadEnvironment
	}
	return ne.userData, nil
}

func (ne *nativeEnvironment) GetMetadata() EnvironmentMetadata {
	return EnvironmentMetadata{
		"bundle": ne.options.Bundle,
	}
}

func (ne *nativeEnvironment) Run(request api.ExecRequest) (api.ExecResult, error) {
	if ne.isDead {
		return EmptyExecResult, ErrorDeadEnvironment
	}
	ne.requests <- request
	result := <-ne.results
	return result, nil
}

func (ne *nativeEnvironment) Shutdown() error {
	if ne.isDead {
		return ErrorDeadEnvironment
	}
	ne.control <- 1
	ne.isDead = true
	return nil
}
