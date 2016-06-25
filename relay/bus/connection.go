package bus

import (
	"errors"
)

// MessagePublisher sends messages on the message bus
type MessagePublisher interface {
	Publish(topic string, message []byte) error
}

// Event describes different events which can happen over the
// life of a connection. Currently only BusConnection is supported.
type Event int

const (
	// ConnectedEvent indicates a working bus connection has been
	// established
	ConnectedEvent Event = iota
)

// SubscriptionHandler is called when a message is received on its
// corresponding topic subscription
type SubscriptionHandler func(conn Connection, topic string, message []byte)

// EventHandler is called when a bus event occurs.
type EventHandler func(conn Connection, event Event)

// DisconnectMessage is sent when the connection is broken
type DisconnectMessage struct {
	Topic string
	Body  string
}

// ConnectionOptions describe how to configure a bus.Connection
type ConnectionOptions struct {
	Userid        string
	Password      string
	Host          string
	Port          int
	SSLEnabled    bool
	SSLCertPath   string
	EventsHandler EventHandler
	AutoReconnect bool
	OnDisconnect  *DisconnectMessage
}

// Connection is the high-level message bus interface
type Connection interface {
	Connect(options ConnectionOptions) error
	Disconnect() error
	Publish(topic string, payload []byte) error
	Subscribe(topic string, handler SubscriptionHandler) error
}

var errorBadTLSCert = errors.New("Bad TLS certificate")
