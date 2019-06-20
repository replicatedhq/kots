package types

import (
	"fmt"
	"os"
	"time"
)

type UpdateSession struct {
	ID             string
	WatchID        string
	CreatedAt      time.Time
	FinishedAt     time.Time
	Result         string
	StateJSON      []byte
	UploadURL      string
	UploadSequence int
	UserID         string
}

func (s *UpdateSession) GetID() string {
	return s.ID
}

func (s *UpdateSession) GetWatchID() string {
	return s.WatchID
}

func (s *UpdateSession) GetType() string {
	return "ship-update"
}

func (s *UpdateSession) GetRole() string {
	return "update"
}

func (s *UpdateSession) GetName() string {
	return fmt.Sprintf("shipupdate-%s", s.ID)
}

func (s *UpdateSession) GetShipArgs() []string {
	if s.UserID == "" {
		return []string{
			"update",
		}
	}

	return []string{
		"update",
		"--headed",
	}
}

func (s *UpdateSession) GetUploadURL() string {
	return s.UploadURL
}

func (s *UpdateSession) GetUploadSequence() int {
	return s.UploadSequence
}

func (s *UpdateSession) GetS3Filepath() string {
	return fmt.Sprintf("%s/%d.tar.gz", s.WatchID, s.UploadSequence)
}

func (s *UpdateSession) GetNodeSelector() string {
	return os.Getenv("UPDATE_NODE_SELECTOR")
}

func (s *UpdateSession) GetCPULimit() string {
	return "500m"
}

func (s *UpdateSession) GetCPURequest() string {
	return "100m"
}

func (s *UpdateSession) GetMemoryLimit() string {
	return "500Mi"
}

func (s *UpdateSession) GetMemoryRequest() string {
	return "100Mi"
}
