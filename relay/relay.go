package relay

import (
	"github.com/operable/go-relay/relay/config"
)

// Incoming request or directive
type Incoming struct {
	Config  *config.Config
	Bus     MessageBus
	Topic   string
	Payload []byte
}

// MessageBus is the interface used by worker code to
// publish messages
type MessageBus interface {
	Run() error
	Halt()
	Publish(topic string, payload []byte) error
	DirectiveReplyTo() string
}
