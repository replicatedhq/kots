package types

import (
	"fmt"
	"os"
	"time"
)

type EditSession struct {
	ID             string
	WatchID        string
	CreatedAt      time.Time
	FinishedAt     time.Time
	IsHeadless     bool
	Result         string
	UserID         string
	StateJSON      []byte
	UploadURL      string
	UploadSequence int
	ParentWatchID  *string
	ParentSequence *int
}

func (s *EditSession) GetID() string {
	return s.ID
}

func (s *EditSession) GetWatchID() string {
	return s.WatchID
}

func (s *EditSession) GetParentWatchID() *string {
	return s.ParentWatchID
}

func (s *EditSession) GetParentSequence() *int {
	return s.ParentSequence
}

func (s *EditSession) GetType() string {
	return "ship-edit"
}

func (s *EditSession) GetRole() string {
	return "edit"
}

func (s *EditSession) GetName() string {
	return fmt.Sprintf("shipedit-%s", s.ID)
}

func (s *EditSession) GetShipArgs() []string {
	if s.IsHeadless {
		return []string{
			"edit",
			"--headless",
		}
	}

	return []string{
		"edit",
	}
}

func (s *EditSession) GetUploadURL() string {
	return s.UploadURL
}

func (s *EditSession) GetUploadSequence() int {
	return s.UploadSequence
}

func (s *EditSession) GetS3Filepath() string {
	return fmt.Sprintf("%s/%d.tar.gz", s.WatchID, s.UploadSequence)
}

func (s *EditSession) GetNodeSelector() string {
	return os.Getenv("EDIT_NODE_SELECTOR")
}

func (s *EditSession) GetCPULimit() string {
	return "500m"
}

func (s *EditSession) GetCPURequest() string {
	return "100m"
}

func (s *EditSession) GetMemoryLimit() string {
	return "500Mi"
}

func (s *EditSession) GetMemoryRequest() string {
	return "100Mi"
}
