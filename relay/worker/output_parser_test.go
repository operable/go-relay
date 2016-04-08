package worker

import (
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
	resp := &messages.ExecutionResponse{}
	output := "COGCMD_DEBUG: Testing 123\nabc\n"
	parseOutput([]byte(output), resp, req)
	if string(resp.Body) != "{\"body\":\"abc\n\"}" {
		t.Errorf("Unexpected parseOutput result: %s", resp.Body)
	}
}

func TestDetectJSON(t *testing.T) {
	resp := &messages.ExecutionResponse{}
	output := "COGCMD_INFO: Testing123\nJSON\n{\"foo\": 123}"
	parseOutput([]byte(output), resp, req)
	if string(resp.Body) != "{\"foo\":123}" {
		t.Errorf("Unexpected parseOutput result: %s", resp.Body)
	}
}
