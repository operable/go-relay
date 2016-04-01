package messages

type ImageSpec struct {
	Name   string `json:"name,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type EnsureImagesEnvelope struct {
	ImageList *EnsureImages `json:"ensure_images,omitempty"`
}
type EnsureImages struct {
	Images []ImageSpec `json:"image_list,omitempty"`
}
