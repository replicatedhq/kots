package types

import (
	"fmt"
	"os"
	"time"
)

type UnforkSession struct {
	ID             string
	UpstreamURI    string
	ForkURI        string
	CreatedAt      time.Time
	FinishedAt     time.Time
	Result         string
	UserID         string
	Username       string
	UploadURL      string
	UploadSequence int
	ParentWatchID  *string
	ParentSequence *int
}

func (s *UnforkSession) GetID() string {
	return s.ID
}

func (s *UnforkSession) GetWatchID() string {
	return s.ID
}

func (s *UnforkSession) GetParentWatchID() *string {
	return s.ParentWatchID
}

func (s *UnforkSession) GetParentSequence() *int {
	return s.ParentSequence
}

func (s *UnforkSession) GetType() string {
	return "ship-unfork"
}

func (s *UnforkSession) GetRole() string {
	return "unfork"
}

func (s *UnforkSession) GetName() string {
	return fmt.Sprintf("shipunfork-%s", s.ID)
}

func (s *UnforkSession) GetShipArgs() []string {
	return []string{
		"unfork",
		"--upstream=" + s.UpstreamURI,
		s.ForkURI,
	}
}

func (s *UnforkSession) GetUploadURL() string {
	return s.UploadURL
}

func (s *UnforkSession) GetUploadSequence() int {
	return s.UploadSequence
}

func (s *UnforkSession) GetS3Filepath() string {
	return fmt.Sprintf("%s/%d.tar.gz", s.ID, s.UploadSequence)
}

func (s *UnforkSession) GetNodeSelector() string {
	return os.Getenv("INIT_NODE_SELECTOR")
}

func (s *UnforkSession) GetCPULimit() string {
	return "500m"
}

func (s *UnforkSession) GetCPURequest() string {
	return "100m"
}

func (s *UnforkSession) GetMemoryLimit() string {
	return "500Mi"
}

func (s *UnforkSession) GetMemoryRequest() string {
	return "100Mi"
}
