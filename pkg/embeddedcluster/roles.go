package embeddedcluster

import (
	"context"
	"fmt"
	"sort"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
)

const DEFAULT_CONTROLLER_ROLE_NAME = "controller"

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
		roles = append(roles, DEFAULT_CONTROLLER_ROLE_NAME)
	}

	for _, role := range config.Custom {
		if role.Name != "" {
			roles = append(roles, role.Name)
		}
	}

	return roles, nil
}

// ControllerRoleName determines the name for the 'controller' role
// this might be part of the config, or it might be the default
func ControllerRoleName(ctx context.Context) (string, error) {
	conf, err := ClusterConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster config: %w", err)
	}

	if conf != nil && conf.Controller.Name != "" {
		return conf.Controller.Name, nil
	}
	return DEFAULT_CONTROLLER_ROLE_NAME, nil
}

// sort roles by name, but put controller first
func SortRoles(controllerRole string, inputRoles []string) []string {
	roles := inputRoles

	// determine if the controller role is present
	hasController := false
	controllerIdx := 0
	for idx, role := range roles {
		if role == controllerRole {
			hasController = true
			controllerIdx = idx
			break
		}
	}

	// if the controller role is present, remove it
	if hasController {
		roles = append(roles[:controllerIdx], roles[controllerIdx+1:]...)
	}

	// sort the roles
	sort.Strings(roles)

	// if the controller role was present, add it back to the front
	if hasController {
		roles = append([]string{controllerRole}, roles...)
	}

	return roles
}
