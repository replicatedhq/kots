package types

import (
	"time"

	"github.com/replicatedhq/ship/pkg/api"
)

// should match https://github.com/replicatedhq/kotsadm/blob/master/api/src/license/license.ts
type License struct {
	ID              string                 `json:"id" yaml:"id" hcl:"id"`
	Assignee        string                 `json:"assignee" yaml:"assignee" hcl:"assignee"`
	CreatedAt       time.Time              `json:"createdAt" yaml:"createdAt" hcl:"createdAt"`
	ExpiresAt       time.Time              `json:"expiresAt" yaml:"expiresAt" hcl:"expiresAt"`
	Type            string                 `json:"type" yaml:"type" hcl:"type"`
	Channel         string                 `json:"channel,omitempty" yaml:"channel,omitempty" hcl:"channel,omitempty"`
	Entitlements    []api.EntitlementValue `json:"entitlements,omitempty" yaml:"entitlements,omitempty" hcl:"entitlements,omitempty"`
	EntitlementSpec string                 `json:"entitlementSpec,omitempty" yaml:"entitlementSpec,omitempty" hcl:"entitlementSpec,omitempty"`
}
