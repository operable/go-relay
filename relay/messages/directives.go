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

// BundleRef is a lightweight record describing the bundle name
// and version installed on a Relay
type BundleRef struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// AnnouncementEnvelope is a wrapper around an Announcement directive.
type AnnouncementEnvelope struct {
	Announcement *Announcement `json:"announce" valid:"required"`
}

// Announcement describes the online/offline status of a Relay
type Announcement struct {
	ID      string      `json:"announcement_id,omitempty" valid:"-"`
	RelayID string      `json:"relay" valid:"required"`
	Online  bool        `json:"online" valid:"bool,required"`
	Bundles []BundleRef `json:"bundles,omitempty"`
	// Deprecated
	Snapshot bool   `json:"snapshot" valid:"bool,required"`
	ReplyTo  string `json:"reply_to,omitempty" valid:"-"`
}

// AnnouncementReceipt is sent by Cog to acknowledge a Relay's bundle announcement
type AnnouncementReceipt struct {
	ID      string      `json:"announcement_id" valid:"-"`
	Status  string      `json:"status" valid:"-"`
	Bundles interface{} `json:"bundles" valid:"-"`
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
func NewBundleAnnouncement(relayID string, bundles []config.Bundle) *AnnouncementEnvelope {
	refs := make([]BundleRef, len(bundles))
	for i, v := range bundles {
		refs[i].Name = v.Name
		refs[i].Version = v.Version
	}
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			RelayID:  relayID,
			Online:   true,
			Bundles:  refs,
			Snapshot: true,
		},
	}
}

// NewBundleAnnouncementExtended builds an Announcement directive describing
// the list of bundles available on a Relay
func NewBundleAnnouncementExtended(relayID string, bundles []config.Bundle, replyTo string, id string) *AnnouncementEnvelope {
	refs := make([]BundleRef, len(bundles))
	for i, v := range bundles {
		refs[i].Name = v.Name
		refs[i].Version = v.Version
	}
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			ID:       id,
			RelayID:  relayID,
			Online:   true,
			Bundles:  refs,
			Snapshot: true,
			ReplyTo:  replyTo,
		},
	}
}
