package policy

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
)

func appSlugFromAppIDGetter(kotsStore store.Store, vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return map[string]string{}, nil
	}
	appIDOrSlug, _ := vars["appId"] // app slug is app ID in Helm
	if appIDOrSlug == "" {
		return map[string]string{}, nil
	}

	app, err := kotsStore.GetApp(appIDOrSlug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}

func appSlugFromSupportbundleGetter(kotsStore store.Store, vars map[string]string) (map[string]string, error) {
	if appSlug, _ := vars["appSlug"]; appSlug != "" {
		return nil, nil
	}
	appIDOrSlug := "" // app slug is app ID in Helm
	if bundleID, _ := vars["bundleId"]; bundleID != "" {
		supportbundle, err := kotsStore.GetSupportBundle(bundleID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from id")
		}
		appIDOrSlug = supportbundle.AppID
	} else if bundleSlug, _ := vars["bundleSlug"]; bundleSlug != "" {
		supportbundle, err := kotsStore.GetSupportBundle(bundleSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get support bundle from slug")
		}
		appIDOrSlug = supportbundle.AppID
	} else {
		return nil, nil
	}

	app, err := kotsStore.GetApp(appIDOrSlug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	return map[string]string{
		"appSlug": app.Slug,
	}, nil
}
