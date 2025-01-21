package types

import (
	"time"

	"github.com/gorilla/websocket"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/pkg/errors"
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

func (m Message) Validate() error {
	if m.AppSlug == "" {
		return errors.New("app slug is missing")
	}
	if m.VersionLabel == "" {
		return errors.New("version label is missing")
	}
	if m.StepID == "" {
		return errors.New("step ID is missing")
	}
	if err := m.Command.Validate(); err != nil {
		return err
	}
	return nil
}

type Command string

const (
	CommandUpgradeCluster   Command = "upgrade-cluster"
	CommandUpgradeManager   Command = "upgrade-manager"
	CommandAddExtension     Command = "add-extension"
	CommandUpgradeExtension Command = "upgrade-extension"
	CommandRemoveExtension  Command = "remove-extension"
)

func (c Command) Validate() error {
	switch c {
	case CommandUpgradeCluster, CommandUpgradeManager, CommandAddExtension, CommandUpgradeExtension, CommandRemoveExtension:
		return nil
	case "":
		return errors.New("command is missing")
	default:
		return errors.Errorf("unknown command: %s", c)
	}
}

type UpgradeManagerData struct {
	LicenseID       string `json:"licenseID"`
	LicenseEndpoint string `json:"licenseEndpoint"`
}

func (d UpgradeManagerData) Validate() error {
	if d.LicenseID == "" {
		return errors.New("license ID is missing")
	}
	if d.LicenseEndpoint == "" {
		return errors.New("license endpoint is missing")
	}
	return nil
}

type UpgradeClusterData struct {
	Installation ecv1beta1.Installation `json:"installation"`
}

func (d UpgradeClusterData) Validate() error {
	if d.Installation.Name == "" {
		return errors.New("installation is missing")
	}
	return nil
}

type ExtensionData struct {
	Repos []k0sv1beta1.Repository `json:"repos"`
	Chart ecv1beta1.Chart         `json:"chart"`
}

func (d ExtensionData) Validate() error {
	if d.Chart.Name == "" {
		return errors.New("chart is missing")
	}
	return nil
}
