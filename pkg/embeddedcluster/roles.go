package embeddedcluster

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
)

const DEFAULT_CONTROLLER_ROLE_NAME = "controller"

var labelValueRegex = regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)

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

// getRoleNodeLabels looks up roles in the cluster config and determines the additional labels to be applied from that
func getRoleNodeLabels(ctx context.Context, roles []string) ([]string, error) {
	config, err := ClusterConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	return getRoleLabelsImpl(config, roles), nil
}

func labelify(s string) string {
	// remove illegal characters
	removechars := labelValueRegex.ReplaceAllString(s, "-")
	// remove leading dashes
	trimmed := strings.TrimPrefix(removechars, "-")
	// restrict it to 63 characters
	if len(trimmed) > 63 {
		trimmed = trimmed[:63]
	}
	// remove trailing dashes
	trimmed = strings.TrimSuffix(trimmed, "-")
	return trimmed
}

func getRoleLabelsImpl(config *embeddedclusterv1beta1.ConfigSpec, roles []string) []string {
	toReturn := []string{}

	if config == nil {
		return toReturn
	}

	for _, role := range roles {
		if role == config.Controller.Name {
			for k, v := range config.Controller.Labels {
				toReturn = append(toReturn, fmt.Sprintf("%s=%s", labelify(k), labelify(v)))
			}
		}
		for _, customRole := range config.Custom {
			if role == customRole.Name {
				for k, v := range customRole.Labels {
					toReturn = append(toReturn, fmt.Sprintf("%s=%s", labelify(k), labelify(v)))
				}
			}
		}
	}

	sort.Strings(toReturn)

	return toReturn
}
