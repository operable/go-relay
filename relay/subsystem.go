package relay

type Subsystem interface {
	Run() error
	Halt() error
}

type DisabledSubsystem struct {
}

func (ds *DisabledSubsystem) Run() error {
	return nil
}

func (ds *DisabledSubsystem) Halt() error {
	return nil
}
