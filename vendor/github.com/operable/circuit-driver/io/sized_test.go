package io

import (
	"bytes"
	"encoding/gob"
	"io"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type testData struct {
	TextField string
}

func TestWriteRead(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	count, err := writer.Write([]byte("hello"))
	if count != 5 {
		t.Errorf("Expected write length of 5: %d", count)
	}
	if err != nil {
		t.Error(err)
	}
	readData := make([]byte, 5)
	count, err = reader.Read(readData)
	if err != nil {
		t.Error(err)
	}
	if count != 5 {
		t.Errorf("Expected read length of 5: %d", count)
	}
}

func TestGobWriteRead(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	tdIn := testData{
		TextField: "a/b/c/d",
	}
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(tdIn); err != nil {
		t.Error(err)
	}
	decoder := gob.NewDecoder(reader)
	tdOut := testData{}
	if err := decoder.Decode(&tdOut); err != nil && err != io.EOF {
		t.Error(err)
	}
	if tdIn.TextField != tdOut.TextField {
		t.Errorf("Input doesn't match output! In: %s; Out: %s",
			tdIn.TextField, tdOut.TextField)
	}
}

func TestGobMultiWriteRead(t *testing.T) {
	tds := makeTestData(5)
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	encoder := gob.NewEncoder(writer)
	for _, v := range tds {
		err := encoder.Encode(v)
		if err != nil {
			t.Fatal(err)
		}
	}
	decoder := gob.NewDecoder(reader)
	for i := 0; i > len(tds); i++ {
		tdOut := testData{}
		if err := decoder.Decode(&tdOut); err != nil && err != io.EOF {
			t.Fatal(err)
		}
		compare(tds[i], tdOut, t, i)
	}
}

func TestInterleavedWriteRead(t *testing.T) {
	tds := makeTestData(5)
	var buf bytes.Buffer
	writer := NewCircuitWriter(&buf)
	reader := NewCircuitReader(&buf)
	decoder := gob.NewDecoder(reader)
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(tds[0]); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Encode(tds[1]); err != nil {
		t.Fatal(err)
	}
	tdIn := testData{}
	if err := decoder.Decode(&tdIn); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	compare(tds[0], tdIn, t, 0)
	if err := encoder.Encode(tds[2]); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&tdIn); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	compare(tds[1], tdIn, t, 1)
	if err := decoder.Decode(&tdIn); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	compare(tds[2], tdIn, t, 2)
}

func makeTestData(count int) []testData {
	retval := make([]testData, 5)
	for i := 0; i < count; i++ {
		retval[i] = testData{
			TextField: randomString(15),
		}
	}
	return retval
}

func compare(tdIn, tdOut testData, t *testing.T, index int) {
	if tdIn.TextField != tdOut.TextField {
		t.Errorf("testData at index %d doesn't match.\nExpected: %+v\nReceived: %+v",
			index, tdIn, tdOut)
	}
}

func randomString(max int) string {
	min := int(max/2) + 1
	size := rand.Intn((max - min)) + min
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(rand.Intn(25) + 97)
	}
	return string(buf)
}
