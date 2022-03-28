package types

import (
	"time"

	"github.com/blang/semver"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

type AppVersion struct {
	KOTSKinds    *kotsutil.KotsKinds `json:"kotsKinds"`
	AppID        string              `json:"appId"`
	Sequence     int64               `json:"sequence"`
	UpdateCursor string              `json:"updateCursor"`
	ChannelID    string              `json:"channelId"`
	VersionLabel string              `json:"versionLabel"`
	Status       string              `json:"status"`
	CreatedOn    time.Time           `json:"createdOn"`
	DeployedAt   *time.Time          `json:"deployedAt"`

	Semver *semver.Version `json:"-"`
	Cursor *cursor.Cursor  `json:"-"`
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
