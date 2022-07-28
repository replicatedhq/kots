package types

import (
	"time"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

type ByCreated []*SupportBundle

func (a ByCreated) Len() int           { return len(a) }
func (a ByCreated) Less(i, j int) bool { return a[i].CreatedAt.Before(a[j].CreatedAt) }
func (a ByCreated) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type SupportBundle struct {
	ID         string                `json:"id"`
	Slug       string                `json:"slug"`
	AppID      string                `json:"appId"`
	Name       string                `json:"name"`
	Size       float64               `json:"size"`
	Status     SupportBundleStatus   `json:"status"`
	TreeIndex  string                `json:"treeIndex,omitempty"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  *time.Time            `json:"updatedAt"`
	UploadedAt *time.Time            `json:"uploadedAt"`
	SharedAt   *time.Time            `json:"sharedAt"`
	IsArchived bool                  `json:"isArchived"`
	Progress   SupportBundleProgress `json:"progress"`
	URI        string                `json:"uri"`
	RedactURIs []string              `json:"redactURIs"`

	BundleSpec          *troubleshootv1beta2.SupportBundle `json:"-"`
	AdditionalRedactors *troubleshootv1beta2.Redactor      `json:"-"`
	Analysis            *SupportBundleAnalysis             `json:"-"`
}

// TODO(dan): analyzer progress
type SupportBundleProgress struct {
	CollectorCount      int    `json:"collectorCount"`
	CollectorsCompleted int    `json:"collectorsCompleted"`
	Message             string `json:"message"`
}

type SupportBundleStatus string

const (
	BUNDLE_FAILED   SupportBundleStatus = "failed"
	BUNDLE_UPLOADED SupportBundleStatus = "uploaded"
	BUNDLE_RUNNING  SupportBundleStatus = "running"
)

type SupportBundleAnalysis struct {
	Insights  []SupportBundleInsight `json:"insights"`
	CreatedAt time.Time              `json:"createdAt"`
}

type SupportBundleInsight struct {
	Key             string                  `json:"key"`
	Severity        string                  `json:"severity"`
	Primary         string                  `json:"primary"`
	Detail          string                  `json:"detail"`
	Icon            string                  `json:"icon"`
	IconKey         string                  `json:"iconKey"`
	DesiredPosition float64                 `json:"desiredPosition"`
	InvolvedObject  *corev1.ObjectReference `json:"involvedObject"`
}

type FileTree struct {
	Nodes []FileTreeNode `json:",inline"`
}

type FileTreeNode struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Children []FileTreeNode `json:"children,omitempty"`
}

type TroubleshootOptions struct {
	Origin        string
	InCluster     bool
	DisableUpload bool
}
