package deploy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	plantypes "github.com/replicatedhq/kots/pkg/plan/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/upgradeservice/plan"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

type CanDeployEC2Options struct {
	Params           types.UpgradeServiceParams
	KotsKinds        *kotsutil.KotsKinds
	RegistrySettings registrytypes.RegistrySettings
}

func CanDeployEC2(opts CanDeployEC2Options) (bool, string, error) {
	needsConfig, err := kotsadmconfig.NeedsConfiguration(
		opts.Params.AppSlug,
		opts.Params.NextSequence,
		opts.Params.AppIsAirgap,
		opts.KotsKinds,
		opts.RegistrySettings,
	)
	if err != nil {
		return false, "", errors.Wrap(err, "check if version needs configuration")
	}
	if needsConfig {
		return false, "cannot deploy because version needs configuration", nil
	}

	pd, err := upgradepreflight.GetPreflightData()
	if err != nil {
		return false, "", errors.Wrap(err, "get preflight data")
	}
	if pd.Result != nil && pd.Result.HasFailingStrictPreflights {
		return false, "cannot deploy because a strict preflight check has failed", nil
	}

	return true, "", nil
}

type DeployEC2Options struct {
	Ctx                          context.Context
	IsSkipPreflights             bool
	ContinueWithFailedPreflights bool
	Params                       types.UpgradeServiceParams
	KotsKinds                    *kotsutil.KotsKinds
	RegistrySettings             registrytypes.RegistrySettings
}

func DeployEC2(opts DeployEC2Options) error {
	// put the app version archive in the object store so the operator
	// of the new kots version can retrieve it to deploy the app
	tgzArchiveKey := fmt.Sprintf(
		"deployments/%s/%s-%s.tar.gz",
		opts.Params.AppSlug,
		opts.Params.UpdateChannelID,
		opts.Params.UpdateCursor,
	)
	if err := apparchive.CreateAppVersionArchive(opts.Params.AppArchive, tgzArchiveKey); err != nil {
		return errors.Wrap(err, "create app version archive")
	}

	preflightData, err := upgradepreflight.GetPreflightData()
	if err != nil {
		return errors.Wrap(err, "get preflight data")
	}

	preflightResult := ""
	if preflightData.Result != nil {
		preflightResult = preflightData.Result.Result
	}

	stepOutput, err := json.Marshal(map[string]string{
		"app-version-archive":             tgzArchiveKey,
		"source":                          opts.Params.Source,
		"skip-preflights":                 fmt.Sprintf("%t", opts.IsSkipPreflights),
		"continue-with-failed-preflights": fmt.Sprintf("%t", opts.ContinueWithFailedPreflights),
		"preflight-result":                preflightResult,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	if err := plan.UpdateStepStatus(opts.Params, plantypes.StepStatusComplete, "", string(stepOutput)); err != nil {
		return errors.Wrap(err, "update step status")
	}

	return nil
}
