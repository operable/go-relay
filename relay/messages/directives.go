package messages

import (
	"github.com/operable/go-relay/relay/config"
)

// ListBundlesEnvelope is a wrapper around a ListBundles directive.
type ListBundlesEnvelope struct {
	ListBundles *ListBundlesMessage `json:"list_bundles"`
}

// ListBundlesMessage asks Cog to send the complete list of
// bundles which should be installed on a given Relay
type ListBundlesMessage struct {
	RelayID string `json:"relay_id"`
	ReplyTo string `json:"reply_to"`
}

// ListBundlesResponseEnvelope is a wrapper around
// the response to a ListBundles directive.
type ListBundlesResponseEnvelope struct {
	Bundles []BundleSpec `json:"bundles"`
}

// BundleSpec describes a command bundle and its current
// enabled status
type BundleSpec struct {
	Name       string        `json:"name,omitempty"`
	Enabled    bool          `json:"enabled,omitempty"`
	ConfigFile config.Bundle `json:"config_file,omitempty"`
}

// AnnouncementEnvelope is a wrapper around an Announcement directive.
type AnnouncementEnvelope struct {
	Announcement *Announcement `json:"announce" valid:"required"`
}

// Announcement describes the online/offline status of a Relay
type Announcement struct {
	RelayID string       `json:"relay" valid:"required"`
	Online  bool         `json:"online" valid:"bool,required"`
	Bundles []BundleSpec `json:"bundles,omitempty"`
	// Deprecated
	Snapshot bool `json:"snapshot" valid:"bool,required"`
}

// NewAnnouncement builds an Announcement directive suitable for
// publishing
func NewAnnouncement(relayID string, online bool) *AnnouncementEnvelope {
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			RelayID:  relayID,
			Online:   online,
			Snapshot: true,
		},
	}
}

// NewBundleAnnouncement builds an Announcement directive describing
// the list of bundles available on a Relay
func NewBundleAnnouncement(relayID string, bundles []string) *AnnouncementEnvelope {
	specs := make([]BundleSpec, len(bundles))
	for i, v := range bundles {
		specs[i].Name = v
	}
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			RelayID:  relayID,
			Online:   true,
			Bundles:  specs,
			Snapshot: true,
		},
	}
}
