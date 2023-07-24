package deploy

import (
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func IsEnabled(identitySpec *kotsv1beta1.Identity, identityConfig *kotsv1beta1.IdentityConfig) bool {
	return identitySpec != nil && identityConfig != nil && identityConfig.Spec.Enabled
}
