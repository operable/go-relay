package worker

import (
	"encoding/json"
	"github.com/operable/go-relay/relay/messages"
	"testing"
)

var (
	req = messages.ExecutionRequest{
		Command: "foo:bar",
		ReplyTo: "/bot/pipelines/123456/reply",
		Requestor: messages.ChatUser{
			ID:       "U0123123",
			Handle:   "jondoe",
			Provider: "slack",
		},
		User: messages.CogUser{
			ID:        "123123123",
			Email:     "jondoe@anonymous.org",
			FirstName: "Jon",
			LastName:  "Doe",
			Username:  "jondoe",
		},
	}
)

func TestParseLogOutput(t *testing.T) {
	req.Parse()
	resp := &messages.ExecutionResponse{}
	output := "COGCMD_DEBUG: Testing 123\nabc\n"
	parseOutput([]byte(output), []byte{}, nil, resp, req)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "[{\"body\":\"abc\\n\"}]" {
		t.Errorf("Unexpected parseOutput result: [{\"body\":\"abc\\n\"}] != %s", string(text))
	}
}

func TestDetectJSON(t *testing.T) {
	req.Parse()
	resp := &messages.ExecutionResponse{}
	output := "COGCMD_INFO: Testing123\nJSON\n{\"foo\": 123}"
	parseOutput([]byte(output), []byte{}, nil, resp, req)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "{\"foo\":123}" {
		t.Errorf("Unexpected parseOutput result: %s", text)
	}
}
