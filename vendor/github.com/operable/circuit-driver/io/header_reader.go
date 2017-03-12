package io

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
)

var EOF = io.EOF

func CircuitHeaderParser(reader io.Reader) (uint32, bool, error) {
	prefix := make([]byte, 4)
	count, err := io.ReadFull(reader, prefix)
	if err != nil && count == 0 {
		return 0, false, err
	}
	size := binary.BigEndian.Uint32(prefix)
	return size, false, nil
}

func DockerStdoutHeaderParser(reader io.Reader) (uint32, bool, error) {
	return (dockerHeaderParser(1, reader))
}

func DockerStderrHeaderParser(reader io.Reader) (uint32, bool, error) {
	return (dockerHeaderParser(2, reader))
}

func dockerHeaderParser(streamID byte, reader io.Reader) (uint32, bool, error) {
	prefix := make([]byte, 8)
	count, err := io.ReadFull(reader, prefix)
	if err != nil && count == 0 {
		return 0, false, err
	}
	discardData := prefix[0] != streamID
	if prefix[1] == 0 && prefix[2] == 0 && prefix[3] == 0 {
		size := binary.BigEndian.Uint32(prefix[4:])
		return size, discardData, nil
	}
	return 0, discardData, errors.New("Corrupt Docker stream header detected")
}

// HeaderParser is a function which reads headers and returns
// the data segment length.
type HeaderParser func(io.Reader) (uint32, bool, error)

// HeaderReader reads packets with headers
type HeaderReader struct {
	reader io.Reader
	buffer *bytes.Buffer
	parser HeaderParser
}

func NewCircuitReader(reader io.Reader) io.Reader {
	return newHeaderReader(reader, CircuitHeaderParser)
}

func NewDockerStdoutReader(reader io.Reader) io.Reader {
	return newHeaderReader(reader, DockerStdoutHeaderParser)
}

func NewDockerStderrReader(reader io.Reader) io.Reader {
	return newHeaderReader(reader, DockerStderrHeaderParser)
}

func (hr HeaderReader) Read(p []byte) (int, error) {
	if hr.buffer.Len() == 0 {
		count, err := hr.refill()
		if err != nil && err != io.EOF {
			return count, err
		}
	}
	count, err := hr.buffer.Read(p)
	return count, err
}

func (hr HeaderReader) refill() (int, error) {
	for {
		packetSize, discard, err := hr.parser(hr.reader)
		if err != nil && packetSize == 0 {
			return 0, err
		}
		if discard == false {
			return hr.fillBuffer(packetSize)
		}
		hr.discard(packetSize)
	}

}

func newHeaderReader(reader io.Reader, parser HeaderParser) io.Reader {
	return HeaderReader{
		reader: reader,
		buffer: bytes.NewBuffer([]byte{}),
		parser: parser,
	}
}

func (hr HeaderReader) discard(packetSize uint32) {
	io.CopyN(ioutil.Discard, hr.reader, int64(packetSize))
}

func (hr HeaderReader) fillBuffer(packetSize uint32) (int, error) {
	count, err := io.CopyN(hr.buffer, hr.reader, int64(packetSize))
	return int(count), err
}

func (hr HeaderReader) readAll(p []byte) (int, error) {
	bytesRead := 0
	for bytesRead < len(p) {
		count, err := hr.reader.Read(p[bytesRead:])
		bytesRead += count
		if err != nil {
			return bytesRead, err
		}
	}
	return bytesRead, nil
}
