package engines

// Engine defines the execution engine interface
type Engine interface {
	IsAvailable(interface{}) (bool, error)
	Execute(interface{}) ([]byte, error)
}
