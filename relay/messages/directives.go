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

// BundleSpec is just a reference to a parsed config file.
type BundleSpec struct {
	ConfigFile config.Bundle `json:"config_file,omitempty"`
}

// BundleRef is a lightweight record describing the bundle name
// and version installed on a Relay
type BundleRef struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// GetDynamicConfigsEnvelope is a wrapper around a GetDynamicConfigs directive.
type GetDynamicConfigsEnvelope struct {
	GetDynamicConfigs *GetDynamicConfigs `json:"get_dynamic_configs"`
}

// GetDynamicConfigs asks Cog to send the complete list of
// dynamic configs for the bundles assigned to the Relay.
type GetDynamicConfigs struct {
	RelayID   string `json:"relay_id"`
	Signature string `json:"config_hash"`
	ReplyTo   string `json:"reply_to"`
}

// DynamicConfigsResponseEnvelope is a wrapper around the
// response to a GetDynamicConfigs directive.
//
// Configs is a map of bundle name to a list of all config layers for
// that bundle.
type DynamicConfigsResponseEnvelope struct {
	Signature string                     `json:"signature"`
	Changed   bool                       `json:"changed"`
	Configs   map[string][]DynamicConfig `json:"configs"`
}

// DynamicConfig is the contents of a dynamic config layer file for a
// bundle.
type DynamicConfig struct {
	Layer      string      `json:"layer"`
	Name       string      `json:"name"`
	Config     interface{} `json:"config"`
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

// NewOfflineAnnouncement builds an Announcement informing Cog the Relay is offline
func NewOfflineAnnouncement(relayID string, replyTo string) *AnnouncementEnvelope {
	return &AnnouncementEnvelope{
		Announcement: &Announcement{
			ID:       "0",
			RelayID:  relayID,
			Online:   false,
			Snapshot: true,
			ReplyTo:  replyTo,
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
