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
