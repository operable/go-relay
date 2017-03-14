package api

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	epl := WrapEncoder(&buf)
	td := NewExecRequest()
	td.SetExecutable("/bin/sayit")
	err := epl.EncodeRequest(td)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Serialized to %d bytes\n", buf.Len())
	data := buf.Bytes()
	if len(data) < 5 {
		t.Errorf("Unexpected data size: %d", len(data))
	}
}

func TestDecode(t *testing.T) {
	var buf bytes.Buffer
	var requestOut ExecRequest
	requestIn := NewExecRequest()
	epl := WrapEncoder(&buf)
	requestIn.SetExecutable("/bin/sayit")
	requestIn.PutEnv("FOO", "123")
	if err := epl.EncodeRequest(requestIn); err != nil {
		t.Error(err)
	}
	dpl := WrapDecoder(&buf)
	err := dpl.DecodeRequest(&requestOut)
	if err != nil {
		t.Error(err)
	}
	if requestOut.GetExecutable() != requestIn.GetExecutable() {
		t.Errorf("Unexpected Executable value: %s", requestOut.Executable)
	}
	if len(requestOut.Env) != 1 {
		t.Errorf("Unexpected env length: %d", len(requestOut.Env))
	}
	if requestOut.FindEnv("FOO") != "123" {
		t.Errorf("Unexpected payload: %v", requestOut.Env)
	}
}
