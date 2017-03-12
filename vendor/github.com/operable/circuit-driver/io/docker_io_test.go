package io

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"
)

func TestSmallDockerWriteRead(t *testing.T) {
	var buf bytes.Buffer
	writer := NewDockerStderrWriter(&buf)
	reader := NewDockerStderrReader(&buf)
	testPhrase := "this is a test"
	count, err := writer.Write([]byte(testPhrase))
	if err != nil {
		t.Fatal(err)
	}
	if count != len(testPhrase) {
		t.Errorf("Expected write count of %d: %d", len(testPhrase), count)
	}
	outBuf := make([]byte, len(testPhrase))
	count, err = reader.Read(outBuf)
	if err != nil {
		t.Errorf("Read error: %s", err)
	}
	if count != len(testPhrase) {
		t.Errorf("Expected read count of %d: %d", len(testPhrase), count)
	}
	if string(outBuf) != testPhrase {
		t.Errorf("Input != output: Input: %s; Output: %s", testPhrase, string(outBuf))
	}
}

func TestBigDockerWriteRead(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCircuitWriter(NewDockerStderrWriter(&buf))
	reader := NewCircuitReader(NewDockerStderrReader(&buf))
	encoder := gob.NewEncoder(writer)
	decoder := gob.NewDecoder(reader)
	now := time.Now()
	err := encoder.Encode(now)
	if err != nil {
		t.Fatal(err)
	}
	var timeOut time.Time
	err = decoder.Decode(&timeOut)
	if err != nil {
		t.Fatal(err)
	}
	if now != timeOut {
		t.Errorf("Input != output: Input: %v; Output: %v", now, timeOut)
	}
}
