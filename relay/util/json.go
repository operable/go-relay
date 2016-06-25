package util

import (
	"encoding/json"
	"io"
)

func NewJsonDecoder(reader io.Reader) *json.Decoder {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder
}
