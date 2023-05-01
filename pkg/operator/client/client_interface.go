package client

import (
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
)

type ClientInterface interface {
	Init() error
	Shutdown()
	DeployApp(deployArgs operatortypes.DeployAppArgs) (deployed bool, finalError error)
	UndeployApp(undeployArgs operatortypes.UndeployAppArgs) (undeployed bool, finalError error)
	ApplyAppInformers(args operatortypes.AppInformersArgs)
}
