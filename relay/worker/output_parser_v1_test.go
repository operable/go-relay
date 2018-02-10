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
var outputParser = NewOutputParserV1()

func TestParseLogOutput(t *testing.T) {
	req.Parse()
	result := api.ExecResult{
		Stdout: []byte("COGCMD_DEBUG: Testing 123\nabc\n123\n"),
		Stderr: emptyStream,
	}
	result.SetSuccess(true)
	resp := outputParser.Parse(result, req, nil)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "[{\"body\":[\"abc\",\"123\"]}]" {
		t.Errorf("Unexpected parseOutput result: [{\"body\":[\"abc\",\"123\"]}] != %s", string(text))
	}
}

func TestDetectJSON(t *testing.T) {
	req.Parse()
	result := api.ExecResult{
		Stdout: []byte("COGCMD_INFO: Testing123\nJSON\n{\"foo\": 123}"),
		Stderr: emptyStream,
	}
	result.SetSuccess(true)
	resp := outputParser.Parse(result, req, nil)
	text, _ := json.Marshal(resp.Body)
	if string(text) != "{\"foo\":123}" {
		t.Errorf("Unexpected parseOutput result: %s", text)
	}
}

func TestNoOutput(t *testing.T) {
	req.Parse()
	result := api.ExecResult{}
	resp := outputParser.Parse(result, req, nil)
	if resp.Body != nil {
		t.Errorf("Unexpected parseOutput result: %s", resp.Body)
	}
}

func TestNoErrorIfResultIsSuccess(t *testing.T) {
	req.Parse()
	result := api.ExecResult{
		Stdout: []byte("Yay, output"),
		Stderr: []byte("Some supplementary output"),
	}
	result.SetSuccess(true)
	resp := outputParser.Parse(result, req, nil)

	if resp.Status != "ok" {
		t.Errorf("Unexpected response status %s", resp.Status)
	}

	text, _ := json.Marshal(resp.Body)
	if string(text) != "[{\"body\":[\"Yay, output\"]}]" {
		t.Errorf("Unexpected body %s", text)
	}
}

func TestErrorOnlyIfResultIsNotSuccess(t *testing.T) {
	req.Parse()
	result := api.ExecResult{
		Stdout: []byte("Yay, output"),
		Stderr: []byte("Bad stuff happened"),
	}
	result.SetSuccess(false)
	resp := outputParser.Parse(result, req, nil)

	if resp.Status == "ok" {
		t.Errorf("Unexpected response status %s", resp.Status)
	}

	if resp.StatusMessage != "Bad stuff happened" {
		t.Errorf("Unexpected response status message %s", resp.StatusMessage)
	}
}
