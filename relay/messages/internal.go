package messages

type AnnouncementMessage struct {
	Envelope *AnnouncementEnvelope `json:"data" valid:"required"`
}
type AnnouncementEnvelope struct {
	Announcement *Announcement `json:"announce" valid:"required"`
}
type Announcement struct {
	RelayID string `json:"relay" valid:"printableascii,required"`
	Online  bool   `json:"online" valid:"bool,required"`
}

func NewAnnouncement(relayID string, online bool) *AnnouncementMessage {
	return &AnnouncementMessage{
		Envelope: &AnnouncementEnvelope{
			Announcement: &Announcement{
				RelayID: relayID,
				Online:  online,
			},
		},
	}
}
