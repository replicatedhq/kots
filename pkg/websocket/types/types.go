package types

import (
	"time"

	"github.com/gorilla/websocket"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
)

type WSClient struct {
	Conn         *websocket.Conn `json:"-"`
	ConnectedAt  time.Time       `json:"connectedAt"`
	LastPingSent PingPongInfo    `json:"lastPingSent"`
	LastPongRecv PingPongInfo    `json:"lastPongRecv"`
	LastPingRecv PingPongInfo    `json:"lastPingRecv"`
	LastPongSent PingPongInfo    `json:"lastPongSent"`
	Version      string          `json:"version"`
}

type PingPongInfo struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

type Message struct {
	AppSlug      string  `json:"appSlug"`
	VersionLabel string  `json:"versionLabel"`
	StepID       string  `json:"stepID"`
	Command      Command `json:"command"`
	Data         string  `json:"data"`
}

type Command string

const (
	CommandUpgradeCluster   Command = "upgrade-cluster"
	CommandUpgradeManager   Command = "upgrade-manager"
	CommandAddExtension     Command = "add-extension"
	CommandUpgradeExtension Command = "upgrade-extension"
	CommandRemoveExtension  Command = "remove-extension"
)

type UpgradeManagerData struct {
	LicenseID       string `json:"licenseID"`
	LicenseEndpoint string `json:"licenseEndpoint"`
}

type UpgradeClusterData struct {
	Installation ecv1beta1.Installation `json:"installation"`
}

type ExtensionData struct {
	Repos []k0sv1beta1.Repository `json:"repos"`
	Chart ecv1beta1.Chart         `json:"chart"`
}
