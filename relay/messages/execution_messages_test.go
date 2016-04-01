package messages

import "testing"

const (
	EXEC_REQ = `{"user":
		{"username":"kevsmith",
		 "last_name":"Smith",
		 "id":"ec35dcfe-49e5-44ae-8eee-973145c4777a",
		 "first_name":"Kevin",
		 "email_address":"foo@bar.io"},
"room": {"name":"direct",
		 "id":"D0B91LEMA"},
"requestor": {"provider":
			  "slack",
			  "id":"U02CSPYB1",
			  "handle":"kevsmith"},
"command": "foo:bar",
"args": ["thhat"],
"options": {"long_flag": 123,
			"short_flag": "abc"},
"command_config": null,
"reply_to": "/bot/foo",
"body": {"body": "foo"}}`
)

func TestRequestUnmarshal(t *testing.T) {
	req, err := UnmarshalExecutionRequest([]byte(EXEC_REQ))
	if err != nil {
		t.Fatal(err)
	}
	if req.Requestor.Handle != "kevsmith" {
		t.Errorf("Expected requestor handle 'kevsmith: %s", req.Requestor.Handle)
	}
	if len(req.Args) != 1 {
		t.Errorf("Expected 1 arg: %d", len(req.Args))
	}
	if len(req.Options) != 2 {
		t.Errorf("Expected 2 options: %d", len(req.Options))
	}
	if len(req.CommandConfig) != 0 {
		t.Errorf("Expected empty command config: Found %d entries",
			len(req.CommandConfig))
	}
}

func TestResponseMarsal(t *testing.T) {
	resp := ExecutionResponse{
		Template:      "",
		Status:        "ok",
		StatusMessage: "",
	}
	_, err := MarshalExecutionResponse(&resp)
	if err != nil {
		t.Fatal(err)
	}
}
