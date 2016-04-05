package messages

// ListBundlesEnevelope is a wrapper around a ListBundles
// directive. Required due to how Go handles struct serialization
type ListBundlesEnvelope struct {
	ListBundles *ListBundlesMessage `json:"list_bundles"`
}

// ListBundlesMessage asks Cog to send the complete list of
// bundles which should be installed on a given Relay
type ListBundlesMessage struct {
	RelayID string `json:"relay_id"`
	ReplyTo string `json:"reply_to"`
}

type ListBundlesResponseEnvelope struct {
	Bundles []BundleSpec `json:"bundles"`
}

type BundleSpec struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

type AnnouncementEnvelope struct {
	Announcement *Announcement `json:"announce" valid:"required"`
}
type Announcement struct {
	RelayID string `json:"relay" valid:"printableascii,required"`
	Online  bool   `json:"online" valid:"bool,required"`
}

func NewAnnouncement(relayID string, online bool) *AnnouncementEnvelope {
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			RelayID: relayID,
			Online:  online,
		},
	}
}
