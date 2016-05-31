package exec

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"github.com/docker/engine-api/types"
	"github.com/operable/cogexec/messages"
	"time"
)

var readDeadline = time.Duration(500) * time.Millisecond

// ErrorBadDockerHeader indicates a malformed Docker response header
var ErrorBadDockerHeader = errors.New("Bad Docker stream header")

// ContainerConnection communicates with running Docker containers
// via stdin/stdout over the Docker-supplied HijackedResponse
type ContainerConnection struct {
	conn types.HijackedResponse
}

// NewContainerConnection wraps a raw Docker container
// connection with a ContainerConnection instance.
func NewContainerConnection(conn types.HijackedResponse) *ContainerConnection {
	return &ContainerConnection{
		conn: conn,
	}
}

// Receive reads and decodes a ExecCommandResponse
func (cc *ContainerConnection) Receive() (*messages.ExecCommandResponse, error) {
	payload, err := cc.readPayload()
	if err != nil {
		return nil, err
	}
	accum := bytes.NewBuffer(payload)
	response := messages.ExecCommandResponse{}
	for {
		decoder := gob.NewDecoder(accum)
		err := decoder.Decode(&response)
		if err == nil {
			return &response, nil
		}
		cc.conn.Conn.SetReadDeadline(time.Now().Add(readDeadline))
		payload, err := cc.readPayload()
		if err != nil {
			return nil, err
		}
		accum.Write(payload)
	}
}

// Send encodes and sends a ExecCommandRequest
func (cc *ContainerConnection) Send(request *messages.ExecCommandRequest) error {
	encoder := gob.NewEncoder(cc.conn.Conn)
	return encoder.Encode(request)
}

// Close tears down the Docker connection
func (cc *ContainerConnection) Close() {
	cc.conn.Close()
}

func (cc *ContainerConnection) readPayload() ([]byte, error) {
	size, err := cc.payloadLength()
	if err != nil {
		return nil, err
	}
	payload, err := cc.readExactly(int(size))
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (cc *ContainerConnection) payloadLength() (uint32, error) {
	header, err := cc.readExactly(8)
	if err != nil {
		return 0, err
	}
	// Verify header according to Docker API docs
	if header[0] == 1 && header[1] == 0 && header[2] == 0 && header[3] == 0 {
		return binary.BigEndian.Uint32(header[4:]), nil
	}
	return 0, ErrorBadDockerHeader
}

// Reads exactly the specified number of bytes
func (cc *ContainerConnection) readExactly(count int) ([]byte, error) {
	buf := make([]byte, count)
	offset := 0
	for {
		bytesRead, err := cc.conn.Conn.Read(buf[offset:])
		if err != nil {
			return nil, err
		}
		offset += bytesRead
		if offset == count {
			break
		}
	}
	return buf, nil
}
