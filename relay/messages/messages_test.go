package messages

import (
	"encoding/json"
	"regexp"
	"testing"
)

var templatePattern = regexp.MustCompile("\"template\":")

func TestEmptyTemplate(t *testing.T) {
	resp := ExecutionResponse{}
	text, err := json.Marshal(&resp)
	if err != nil {
		t.Fatal(err)
	}
	if templatePattern.Find(text) != nil {
		t.Error("Empty template field included in marshaled output")
	}
}
