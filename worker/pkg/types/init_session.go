package types

import (
	"fmt"
	"os"
	"time"

	"github.com/replicatedhq/ship/pkg/state"
)

type ShipStateMetadata struct {
	V1 ShipStateMetadataV1 `json:"v1"`
}

type ShipStateMetadataV1 struct {
	Metadata state.Metadata `json:"metadata"`
}

type InitSession struct {
	ID                   string
	UpstreamURI          string
	RequestedUpstreamURI string
	CreatedAt            time.Time
	FinishedAt           time.Time
	Result               string
	UserID               string
	Username             string
	UploadURL            string
	UploadSequence       int
	ClusterID            string
	GitHubPath           string
}

func (s *InitSession) GetID() string {
	return s.ID
}

func (s *InitSession) GetWatchID() string {
	return s.ID
}

func (s *InitSession) GetType() string {
	return "ship-init"
}

func (s *InitSession) GetRole() string {
	return "init"
}

func (s *InitSession) GetName() string {
	return fmt.Sprintf("shipinit-%s", s.ID)
}

func (s *InitSession) GetShipArgs() []string {
	return []string{
		"init",
		s.UpstreamURI,
		"--rm-asset-dest",
	}
}

func (s *InitSession) GetUploadURL() string {
	return s.UploadURL
}

func (s *InitSession) GetUploadSequence() int {
	return s.UploadSequence
}

func (s *InitSession) GetS3Filepath() string {
	return fmt.Sprintf("%s/%d.tar.gz", s.ID, s.UploadSequence)
}

func (s *InitSession) GetNodeSelector() string {
	return os.Getenv("INIT_NODE_SELECTOR")
}

func (s *InitSession) GetCPULimit() string {
	return "500m"
}

func (s *InitSession) GetCPURequest() string {
	return "100m"
}

func (s *InitSession) GetMemoryLimit() string {
	return "500Mi"
}

func (s *InitSession) GetMemoryRequest() string {
	return "100Mi"
}
