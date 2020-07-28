package types

import (
	"time"
)

type SupportBundle struct {
	ID         string     `json:"id"`
	Slug       string     `json:"slug"`
	AppID      string     `json:"appId"`
	Name       string     `json:"name"`
	Size       float64    `json:"size"`
	Status     string     `json:"status"`
	TreeIndex  string     `json:"treeIndex,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UploadedAt *time.Time `json:"uploadedAt"`
	IsArchived bool       `json:"isArchived"`
}

type SupportBundleAnalysis struct {
	ID          string                 `json:"id"`
	Error       string                 `json:"error"`
	MaxSeverity string                 `json:"maxSeverity"`
	Insights    []SupportBundleInsight `json:"insights"`
	CreatedAt   time.Time              `json:"createdAt"`
}

type SupportBundleInsight struct {
	Key             string  `json:"key"`
	Severity        string  `json:"severity"`
	Primary         string  `json:"primary"`
	Detail          string  `json:"detail"`
	Icon            string  `json:"icon"`
	IconKey         string  `json:"iconKey"`
	DesiredPosition float64 `json:"desiredPosition"`
}

type FileTree struct {
	Nodes []FileTreeNode `json:",inline"`
}

type FileTreeNode struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Children []FileTreeNode `json:"children,omitempty"`
}
