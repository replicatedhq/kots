package replicatedapp

type ReplicatedCursor struct {
	ChannelID   string
	ChannelName string
	Cursor      string
}

func (this ReplicatedCursor) Equal(other ReplicatedCursor) bool {
	if this.ChannelID != "" && other.ChannelID != "" {
		return this.ChannelID == other.ChannelID && this.Cursor == other.Cursor
	}
	return this.ChannelName == other.ChannelName && this.Cursor == other.Cursor
}
