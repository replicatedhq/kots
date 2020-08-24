package types

import (
	"time"
)

type AppVersion struct {
	Sequence     int64      `json:"sequence"`
	UpdateCursor int        `json:"updateCursor"`
	VersionLabel string     `json:"versionLabel"`
	Status       string     `json:"status"`
	CreatedOn    *time.Time `json:"createdOn"`
	ReleaseNotes string     `json:"releaseNotes"`
	DeployedAt   *time.Time `json:"deployedAt"`
}

type RealizedLink struct {
	Title string `json:"title"`
	Uri   string `json:"uri"`
}

type ForwardedPort struct {
	ApplicationURL string `json:"applicationUrl"`
	LocalPort      int    `json:"localPort"`
	ServiceName    string `json:"serviceName"`
	ServicePort    int    `json:"servicePort"`
}
