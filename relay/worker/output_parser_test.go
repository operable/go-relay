package worker

import (
	"encoding/json"
	"github.com/operable/circuit-driver/api"
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

var emptyStream = []byte{}

func TestParseLogOutput(t *testing.T) {
	req.Parse()
	result := api.ExecResult{
		Stdout: []byte("COGCMD_DEBUG: Testing 123\nabc\n123\n"),
		Stderr: emptyStream,
	}
	resp := &messages.ExecutionResponse{}
	parseOutput(result, nil, resp, req)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "[{\"body\":[\"abc\",\"123\"]}]" {
		t.Errorf("Unexpected parseOutput result: [{\"body\":[\"abc\",\"123\"]}] != %s", string(text))
	}
}

func TestDetectJSON(t *testing.T) {
	req.Parse()
	resp := &messages.ExecutionResponse{}
	result := api.ExecResult{
		Stdout: []byte("COGCMD_INFO: Testing123\nJSON\n{\"foo\": 123}"),
		Stderr: emptyStream,
	}
	parseOutput(result, nil, resp, req)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "{\"foo\":123}" {
		t.Errorf("Unexpected parseOutput result: %s", text)
	}
}

func TestNoOutput(t *testing.T) {
	req.Parse()
	resp := &messages.ExecutionResponse{}
	result := api.ExecResult{}
	parseOutput(result, nil, resp, req)
	if resp.Body != nil {
		t.Errorf("Unexpected parseOutput result: %s", resp.Body)
	}
}
