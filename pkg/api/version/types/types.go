package types

import (
	"time"

	"github.com/replicatedhq/kots/pkg/kotsutil"
)

type AppVersion struct {
	KOTSKinds  *kotsutil.KotsKinds `json:"kotsKinds"`
	AppID      string              `json:"appId"`
	Sequence   int64               `json:"sequence"`
	Status     string              `json:"status"`
	CreatedOn  time.Time           `json:"createdOn"`
	DeployedAt *time.Time          `json:"deployedAt"`
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
