package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	plantypes "github.com/replicatedhq/kots/pkg/plan/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
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
		"app-id":                          opts.Params.AppID,
		"app-slug":                        opts.Params.AppSlug,
		"app-version-archive":             tgzArchiveKey,
		"base-sequence":                   fmt.Sprintf("%d", opts.Params.BaseSequence),
		"version-label":                   opts.Params.UpdateVersionLabel,
		"source":                          opts.Params.Source,
		"is-airgap":                       fmt.Sprintf("%t", opts.Params.AppIsAirgap),
		"channel-id":                      opts.Params.UpdateChannelID,
		"update-cursor":                   opts.Params.UpdateCursor,
		"skip-preflights":                 fmt.Sprintf("%t", opts.IsSkipPreflights),
		"continue-with-failed-preflights": fmt.Sprintf("%t", opts.ContinueWithFailedPreflights),
		"preflight-result":                preflightResult,
		"embedded-cluster-version":        opts.Params.UpdateECVersion,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	body, err := json.Marshal(map[string]string{
		"versionLabel": opts.Params.UpdateVersionLabel,
		"status":       string(plantypes.StepStatusComplete),
		"output":       string(stepOutput),
	})
	if err != nil {
		return errors.Wrap(err, "marshal request body")
	}

	url := fmt.Sprintf("http://localhost:3000/api/v1/app/%s/plan/%s", opts.Params.AppSlug, opts.Params.PlanStepID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("send request, status code: %d", resp.StatusCode)
	}

	return nil
}
