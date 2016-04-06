package engines

// Engine defines the execution engine interface
type Engine interface {
	IsAvailable(name string, meta string) (bool, error)
	Execute(interface{}) ([]byte, error)
}
