package relay

import (
	"encoding/json"

	"github.com/asaskevich/govalidator"
)

type ChatRoom struct {
	ID   string `json:"id" valid:"alphanum,required"`
	Name string `json:"name" valid:"required"`
}

type Originator struct {
	ID       string `json:"id" valid:"alphanum,required"`
	Provider string `json:"provider" valid:"required"`
	Handle   string `json:"handle" valid:"required"`
}

type CogUser struct {
	ID        string `json:"id" valid:"uuid,required"`
	FirstName string `json:"first_name" valid:"required"`
	LastName  string `json:"last_name" valid:"required"`
}

type UntypedArray []interface{}
type UntypedValueMap map[string]interface{}

type ExecutionRequest struct {
	Room          *ChatRoom       `json:"room" valid:"required"`
	Requestor     *Originator     `json:"requestor" valid:"required"`
	User          *CogUser        `json:"user" valid:"required"`
	Command       string          `json:"command" valid:"required"`
	Options       UntypedValueMap `json:"options"`
	Args          UntypedArray    `json:"args"`
	CommandConfig UntypedValueMap `json:"command_config,omitempty" valid:"-"`
	CogEnv        UntypedValueMap `json:"cog_env,omitempty" valid:"-"`
}

func ParseExecutionRequest(raw []byte) (*ExecutionRequest, error) {
	request := ExecutionRequest{}
	err := json.Unmarshal(raw, &request)
	if err != nil {
		return nil, err
	}
	_, err = govalidator.ValidateStruct(request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}
