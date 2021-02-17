package policy

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
)

func appSlugFromAppIDGetter(kotsStore store.KOTSStore, vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return map[string]string{}, nil
	}
	appID, _ := vars["appId"]
	if appID == "" {
		return map[string]string{}, nil
	}
	app, err := kotsStore.GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}

func appSlugFromSupportbundleGetter(kotsStore store.KOTSStore, vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return nil, nil
	}
	appID := ""
	if bundleID, _ := vars["bundleId"]; bundleID != "" {
		supportbundle, err := kotsStore.GetSupportBundle(bundleID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from id")
		}
		appID = supportbundle.AppID
	} else if bundleSlug, _ := vars["bundleSlug"]; bundleSlug != "" {
		supportbundle, err := kotsStore.GetSupportBundleFromSlug(bundleSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from slug")
		}
		appID = supportbundle.AppID
	} else {
		return nil, nil
	}
	app, err := kotsStore.GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}
