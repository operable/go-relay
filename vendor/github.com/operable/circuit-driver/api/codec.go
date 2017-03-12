package api

import (
	"bytes"
	"encoding/gob"
	circuit "github.com/operable/circuit-driver/io"
	"io"
)

// Encoder takes arbitrary Go data, gob encodes it,
// and uses protocol.Writer to write the resulting data
type Encoder struct {
	writer  io.Writer
	encoder *gob.Encoder
}

// Encoder configures an Encoder and Writer instance
// around a base io.Writer
func WrapEncoder(w io.Writer) Encoder {
	writer := circuit.NewCircuitWriter(w)
	return Encoder{
		writer:  writer,
		encoder: gob.NewEncoder(writer),
	}
}

// EncodeRequest encodes ExecRequests and writes them to the underlying
// transport via protocol.Writer
func (e Encoder) EncodeRequest(request *ExecRequest) error {
	data, err := request.Marshal()
	if err != nil {
		return err
	}
	_, err = e.writer.Write(data)
	return err
}

// EncodeResult encodes ExecResults and writes them to the underlying
// transport via protcol.Writer
func (e Encoder) EncodeResult(result *ExecResult) error {
	data, err := result.Marshal()
	if err != nil {
		return err
	}
	_, err = e.writer.Write(data)
	return err
}

// Decoder reads data via protocol.Reader, gob decodes the payload,
// and returns the result
type Decoder struct {
	reader  io.Reader
	decoder *gob.Decoder
}

// WrapDecoder configures a Decoder and Reader instance
// around a base io.Reader
func WrapDecoder(r io.Reader) Decoder {
	reader := circuit.NewCircuitReader(r)
	return Decoder{
		reader:  reader,
		decoder: gob.NewDecoder(reader),
	}
}

// DecodeRequest reads and decodes ExecRequests
func (d Decoder) DecodeRequest(request *ExecRequest) error {
	var accum bytes.Buffer
	buf := make([]byte, 4096)
	for {
		count, err := d.reader.Read(buf)
		if count > 0 {
			accum.Write(buf[:count])
		}
		unmarshalErr := request.Unmarshal(accum.Bytes())
		if unmarshalErr != nil && err != nil {
			return err
		}
		if unmarshalErr == nil {
			return nil
		}
	}
}

// DecodeResult reads and decodes ExecResults
func (d Decoder) DecodeResult(result *ExecResult) error {
	var accum bytes.Buffer
	buf := make([]byte, 4096)
	for {
		count, err := d.reader.Read(buf)
		if count > 0 {
			accum.Write(buf[:count])
		}
		unmarshalErr := result.Unmarshal(accum.Bytes())
		if unmarshalErr != nil && err != nil {
			return err
		}
		if unmarshalErr == nil {
			return nil
		}
	}
}
