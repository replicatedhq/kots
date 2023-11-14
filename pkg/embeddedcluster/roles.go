package embeddedcluster

import (
	"context"
	"fmt"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
)

// GetRoles will get a list of role names
func GetRoles(ctx context.Context) ([]string, error) {
	config, err := ClusterConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	if config == nil {
		// use the default nil spec
		config = &embeddedclusterv1beta1.ConfigSpec{}
	}

	// determine role names
	roles := []string{}
	if config.Controller.Name != "" {
		roles = append(roles, config.Controller.Name)
	} else {
		roles = append(roles, "controller")
	}

	for _, role := range config.Custom {
		if role.Name != "" {
			roles = append(roles, role.Name)
		}
	}

	return roles, nil
}
