package io

import (
	"bytes"
	"encoding/binary"
	"io"
)

type HeaderGenerator func(data []byte) ([]byte, error)

// HeaderWriter prepends all writes with the output
// of a HeaderGenerator function.
type HeaderWriter struct {
	writer    io.Writer
	generator HeaderGenerator
}

func CircuitHeaderGenerator(data []byte) ([]byte, error) {
	writeSize := uint32(len(data))
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, writeSize)
	return prefix, nil
}

func DockerStdoutHeaderGenerator(data []byte) ([]byte, error) {
	return (dockerHeaderGenerator(1, data))
}

func DockerStderrHeaderGenerator(data []byte) ([]byte, error) {
	return (dockerHeaderGenerator(2, data))
}

// See https://docs.docker.com/engine/reference/io/docker_remote_io_v1.20
func dockerHeaderGenerator(streamID byte, data []byte) ([]byte, error) {
	header := make([]byte, 8)
	header[0] = streamID
	writeSize := uint32(len(data))
	binary.BigEndian.PutUint32(header[4:], writeSize)
	return header, nil
}

func NewCircuitWriter(writer io.Writer) io.Writer {
	return newWriter(writer, CircuitHeaderGenerator)
}

func NewDockerStdoutWriter(writer io.Writer) io.Writer {
	return newWriter(writer, DockerStdoutHeaderGenerator)
}

func NewDockerStderrWriter(writer io.Writer) io.Writer {
	return newWriter(writer, DockerStderrHeaderGenerator)
}

func newWriter(writer io.Writer, generator HeaderGenerator) io.Writer {
	return HeaderWriter{
		writer:    writer,
		generator: generator,
	}
}

// Write is required by the io.Writer interface
func (hw HeaderWriter) Write(p []byte) (int, error) {
	header, err := hw.generator(p)
	if err != nil {
		return 0, err
	}
	headerSize := len(header)
	var buf bytes.Buffer
	buf.Grow(headerSize + len(p))
	buf.Write(header)
	buf.Write(p)
	count, err := hw.writeAll(buf.Bytes())
	if count > len(header) {
		count -= headerSize
	} else {
		count = 0
	}
	return count, err
}

func (hw HeaderWriter) writeAll(p []byte) (int, error) {
	bytesWritten := 0
	for bytesWritten < len(p) {
		count, err := hw.writer.Write(p[bytesWritten:])
		bytesWritten += count
		if err != nil {
			return bytesWritten, err
		}
	}
	return bytesWritten, nil
}
