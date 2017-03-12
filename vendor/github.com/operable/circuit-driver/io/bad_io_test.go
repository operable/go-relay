package io

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type shortReader struct {
	reader io.Reader
}

type shortWriter struct {
	writer io.Writer
}

func (sr shortReader) Read(p []byte) (int, error) {
	if len(p) > 2 {
		pp := make([]byte, int(len(p)/2))
		count, err := sr.reader.Read(pp)
		if err != nil && err != io.EOF {
			return count, err
		}
		for i := 0; i < count; i++ {
			p[i] = pp[i]
		}
		return count, err
	}
	return (sr.reader.Read(p))
}

func (sw shortWriter) Write(p []byte) (int, error) {
	if len(p) > 2 {
		length := int(len(p) / 2)
		return (sw.writer.Write(p[:length]))
	}
	return (sw.writer.Write(p))
}

func TestShortReads(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	count, err := writer.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("Expected write count of 5: %d", count)
	}
	readBack := make([]byte, 5)
	count, err = reader.Read(readBack)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("Expected read count of 5: %d", count)
	}
}

func TestShortWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	inputText := randomString(100)
	count, err := writer.Write([]byte(inputText))
	if err != nil {
		t.Fatal(err)
	}
	if count != len(inputText) {
		t.Errorf("Expected write count of %d: %d", len(inputText), count)
	}
	outputBuf := make([]byte, len(inputText))
	count, err = reader.Read(outputBuf)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if count != len(inputText) {
		t.Errorf("Expected read size of %d: %d", len(inputText), count)
	}
	if string(outputBuf) != inputText {
		t.Errorf("Input doesn't match output: Expected: %s; Received: %s", inputText, string(outputBuf))
	}
}
