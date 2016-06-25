package util

import (
	"encoding/json"
	"io"
)

// NewJSONDecoder creates a new JSON decoder which is configured
// to NOT mangle big integers.
func NewJSONDecoder(reader io.Reader) *json.Decoder {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder
}
