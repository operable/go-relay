package config

import (
	"fmt"
)

// CogInfo contains information required to connect to an upstream Cog host
type CogInfo struct {
	Host            string `yaml:"host" env:"RELAY_COG_HOST" valid:"hostorip,required" default:"127.0.0.1"`
	Port            int    `yaml:"port" env:"RELAY_COG_PORT" valid:"int64,required" default:"1883"`
	Token           string `yaml:"token" env:"RELAY_COG_TOKEN" valid:"required"`
	SSLEnabled      bool   `yaml:"enable_ssl" env:"RELAY_COG_ENABLE_SSL" valid:"bool" default:"false"`
	SSLCertPath     string `yaml:"ssl_cert_path" env:"RELAY_COG_SSL_CERT_PATH" valid:"-"`
	RefreshInterval string `yaml:"refresh_interval" env:"RELAY_COG_REFRESH_INTERVAL" valid:"required" default:"1m"`
}

// URL returns a MQTT URL for the upstream Cog host
func (ci *CogInfo) URL() string {
	proto := "tcp"
	if ci.SSLEnabled {
		proto = "ssl"
	}
	return fmt.Sprintf("%s://%s:%d", proto, ci.Host, ci.Port)
}
