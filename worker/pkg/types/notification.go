package types

import (
	"fmt"
)

type Notification struct {
	ID             string
	WatchID        string
	Type           string
	UploadSequence int
}

func (n *Notification) GetID() string {
	return n.ID
}

func (n *Notification) GetWatchID() string {
	return n.WatchID
}

func (n *Notification) GetType() string {
	return fmt.Sprintf("ship-notification-%s", n.Type)
}

func (n *Notification) GetUploadSequence() int {
	return n.UploadSequence
}

func (n *Notification) GetS3Filepath() string {
	return fmt.Sprintf("%s/%d.tar.gz", n.WatchID, n.UploadSequence)
}
