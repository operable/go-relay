package relay

import "errors"

type Subsystem interface {
	Run() error
	Halt() error
	Invoke(interface{}) (interface{}, error)
}

type CallData struct {
	Caller chan interface{}
	Data   interface{}
}

type DisabledSubsystem struct {
}

func (ds *DisabledSubsystem) Run() error {
	return nil
}

func (ds *DisabledSubsystem) Halt() error {
	return nil
}

func (ds *DisabledSubsystem) Invoke(data interface{}) (interface{}, error) {
	return nil, errors.New("Not implemented")
}
