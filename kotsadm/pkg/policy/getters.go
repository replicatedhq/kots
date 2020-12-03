package policy

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func appSlugFromAppIDGetter(vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return map[string]string{}, nil
	}
	appID, _ := vars["appId"]
	if appID == "" {
		return map[string]string{}, nil
	}
	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}

func appSlugFromSupportbundleGetter(vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return nil, nil
	}
	appID := ""
	if bundleID, _ := vars["bundleId"]; bundleID != "" {
		supportbundle, err := store.GetStore().GetSupportBundle(bundleID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from id")
		}
		appID = supportbundle.AppID
	} else if bundleSlug, _ := vars["bundleSlug"]; bundleSlug != "" {
		supportbundle, err := store.GetStore().GetSupportBundleFromSlug(bundleSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from slug")
		}
		appID = supportbundle.AppID
	} else {
		return nil, nil
	}
	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}
