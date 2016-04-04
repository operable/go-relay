package relay

// Service is any independently running subsystem
type Service interface {
	Run() error
	Halt()
}
