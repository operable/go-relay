package messages

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
	ReplyTo       string          `json:"reply_to" valid:"printableascii,required"`
}

type ExecutionResponse struct {
	Template      string          `json:"template" valid:"printableascii"`
	Status        string          `json:"status" valid:"status,required"`
	StatusMessage string          `json:"status_message" valid:"-"`
	Body          UntypedValueMap `json:"body" valid:"-"`
}

func UnmarshalExecutionRequest(raw []byte) (*ExecutionRequest, error) {
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

func MarshalExecutionResponse(resp *ExecutionResponse) ([]byte, error) {
	govalidator.TagMap["status"] = govalidator.Validator(func(value string) bool {
		return value == "ok" || value == "error"
	})
	_, err := govalidator.ValidateStruct(*resp)
	if err != nil {
		return []byte{}, err
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	return raw, nil
}
