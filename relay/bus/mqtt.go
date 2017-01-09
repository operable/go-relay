package bus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/snappy"
	"io/ioutil"
	"time"
)

// MQTTConnection is a MQTT-specific implementation of
// bus.Connection
type MQTTConnection struct {
	options ConnectionOptions
	conn    *mqtt.Client
	backoff *Backoff
}

// Connect is required by the bus.Connection interface
func (mqc *MQTTConnection) Connect(options ConnectionOptions) error {
	mqttOpts := mqc.buildMQTTOptions(options)
	if err := configureSSL(options, mqttOpts); err != nil {
		return err
	}
	if options.OnDisconnect != nil {
		compressed := snappy.Encode(nil, []byte(options.OnDisconnect.Body))
		mqttOpts.SetWill(options.OnDisconnect.Topic, string(compressed), 1, false)
	}
	mqc.backoff = NewBackoff()
	mqc.conn = mqtt.NewClient(mqttOpts)
	for {
		if token := mqc.conn.Connect(); token.Wait() && token.Error() != nil {
			log.Errorf("Error connecting to %s: %s", brokerURL(options), token.Error())
			mqc.backoff.Wait()
		} else {
			mqc.backoff.Reset()
			break
		}
	}
	mqc.options = options
	if mqc.options.EventsHandler != nil {
		mqc.options.EventsHandler(mqc, ConnectedEvent)
	}
	return nil
}

// Disconnect is required by the bus.Connection interface
func (mqc *MQTTConnection) Disconnect() error {
	mqc.conn.Disconnect(1000)
	return nil
}

// Publish is required by the bus.Connection interface
func (mqc *MQTTConnection) Publish(topic string, payload []byte) error {
	compressed := snappy.Encode(nil, payload)
	token := mqc.conn.Publish(topic, 1, false, compressed)
	token.Wait()
	return token.Error()
}

// Subscribe is required by the bus.Connection interface
func (mqc *MQTTConnection) Subscribe(topic string, handler SubscriptionHandler) error {
	mqttHandler := func(client *mqtt.Client, message mqtt.Message) {
		compressed := message.Payload()
		payload, err := snappy.Decode(nil, compressed)
		if err != nil {
			log.Errorf("Decompressing MQTT payload failed: %s", err)
			return
		}
		handler(mqc, message.Topic(), payload)
	}
	token := mqc.conn.Subscribe(topic, 1, mqttHandler)
	token.Wait()
	return token.Error()
}

func (mqc *MQTTConnection) disconnected(cilent *mqtt.Client, err error) {
	log.Errorf("MQTT connection failed: %s.", err)
	for {
		if token := mqc.conn.Connect(); token.Wait() && token.Error() != nil {
			log.Errorf("Error connecting to %s: %s", brokerURL(mqc.options), token.Error())
			mqc.backoff.Wait()
		} else {
			mqc.backoff.Reset()
			break
		}
	}
	if mqc.options.EventsHandler != nil {
		mqc.options.EventsHandler(mqc, ConnectedEvent)
	}
}

func (mqc *MQTTConnection) buildMQTTOptions(options ConnectionOptions) *mqtt.ClientOptions {
	clientID := fmt.Sprintf("%x", time.Now().UTC().UnixNano())
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.SetAutoReconnect(options.AutoReconnect)
	mqttOpts.SetKeepAlive(time.Duration(60) * time.Second)
	mqttOpts.SetPingTimeout(time.Duration(15) * time.Second)
	mqttOpts.SetUsername(options.Userid)
	mqttOpts.SetPassword(options.Password)
	mqttOpts.SetClientID(clientID)
	mqttOpts.SetCleanSession(true)
	brokerURL := brokerURL(options)
	mqttOpts.AddBroker(brokerURL)
	if !options.AutoReconnect {
		mqttOpts.SetConnectionLostHandler(mqc.disconnected)
	}
	return mqttOpts
}

func configureSSL(options ConnectionOptions, mqttOpts *mqtt.ClientOptions) error {
	if !options.SSLEnabled {
		return nil
	}
	log.Info("SSL enabled on MQTT connection to Cog")
	if options.SSLCertPath == "" {
		log.Warn("TLS certificate verification disabled.")
		mqttOpts.TLSConfig = tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		buf, err := ioutil.ReadFile(options.SSLCertPath)
		if err != nil {
			log.Errorf("Error reading TLS certificate file %s: %s.",
				options.SSLCertPath, err)
			return err
		}
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(buf)
		if !ok {
			log.Errorf("Failed to parse TLS certificate file %s.",
				options.SSLCertPath)
			return errorBadTLSCert
		}
		log.Info("TLS certificate verification enabled.")
		mqttOpts.TLSConfig = tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            roots,
		}
	}
	return nil
}

func brokerURL(options ConnectionOptions) string {
	prefix := "tcp"
	if options.SSLEnabled {
		prefix = "ssl"
	}
	return fmt.Sprintf("%s://%s:%d", prefix, options.Host, options.Port)
}
