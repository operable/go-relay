package api

import (
	"bytes"
	"testing"
)

func TestRequestEncode(t *testing.T) {
	var buf bytes.Buffer
	enc := WrapEncoder(&buf)
	dec := WrapDecoder(&buf)
	var request ExecRequest
	request.SetExecutable("/bin/date")
	var request2 ExecRequest
	enc.EncodeRequest(&request)
	dec.DecodeRequest(&request2)
	if request2.GetExecutable() != request.GetExecutable() {
		t.Errorf("Expected decoded Executable field to be %s: %s", request.GetExecutable(),
			request2.GetExecutable())
	}
	if len(request2.Env) != len(request.Env) {
		t.Errorf("Input env != output: %d; %d", len(request.Env), len(request2.Env))
	}
}
