package relay

import (
	"encoding/json"
	"testing"
)

func TestParsingEnsureImages(t *testing.T) {
	raw := `{"ensure_images": {"image_list": [{"name": "operable/latest"}, {"digest": "53c130b71e1ca2e9000c719777932ab41f2fd8d9"}]}}`
	msg := &EnsureImagesEnvelope{}
	err := json.Unmarshal([]byte(raw), &msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.ImageList.Images) != 2 {
		t.Errorf("Expected image list to contain 2 entries: %d", len(msg.ImageList.Images))
	}
	spec := msg.ImageList.Images[0]
	if spec.Name != "operable/latest" {
		t.Errorf("Expected first image name to be 'operable/latest': %s", spec.Name)
	}
}
