package circuit

import (
	"errors"
	"github.com/docker/docker/client"
	"github.com/operable/circuit-driver/api"
)

type EnvironmentKind int

const (
	NativeKind EnvironmentKind = iota
	DockerKind
)

type EnvironmentMetadata map[string]string
type EnvironmentUserData map[string]interface{}

type Environment interface {
	GetKind() EnvironmentKind
	SetUserData(EnvironmentUserData) error
	GetUserData() (EnvironmentUserData, error)
	GetMetadata() EnvironmentMetadata
	Run(request api.ExecRequest) (api.ExecResult, error)
	Shutdown() error
}

type DockerEnvironmentOptions struct {
	Conn           *client.Client
	Image          string
	Tag            string
	Binds          []string
	DriverInstance string
	DriverPath     string
	Memory         int64
}

type CreateEnvironmentOptions struct {
	Kind          EnvironmentKind
	Bundle        string
	DockerOptions DockerEnvironmentOptions
}

var ErrorDeadEnvironment = errors.New("Dead environment")
var EmptyExecResult = api.ExecResult{}

func CreateEnvironment(options CreateEnvironmentOptions) (Environment, error) {
	switch options.Kind {
	case NativeKind:
		env := &nativeEnvironment{}
		if err := env.init(options); err != nil {
			return nil, err
		}
		return env, nil
	case DockerKind:
		env := &dockerEnvironment{}
		if err := env.init(options); err != nil {
			return nil, err
		}
		return env, nil
	}
	return nil, nil
}
