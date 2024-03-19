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
	if config.Roles.Controller.Name != "" {
		roles = append(roles, config.Roles.Controller.Name)
	} else {
		roles = append(roles, DEFAULT_CONTROLLER_ROLE_NAME)
	}

	for _, role := range config.Roles.Custom {
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

	if conf != nil && conf.Roles.Controller.Name != "" {
		return conf.Roles.Controller.Name, nil
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

// labelKey will clean up a string to be a valid label key.
// ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func labelKey(s string) string {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 1 {
		return labelify(parts[0])
	}
	prefix := ConvertToRFC1123(parts[0])
	name := labelify(parts[1])
	return fmt.Sprintf("%s/%s", prefix, name)
}

// labelify will clean up a string to be a valid label value.
// ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
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

// borrowed from https://github.com/kubernetes/apimachinery/blob/9254095ca5cab3666d500ec67cd00f9ab0d113d7/pkg/util/validation/validation.go#L206-L208

const dns1123LabelFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
const dns1123SubdomainFmt string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"

// DNS1123SubdomainMaxLength is a subdomain's max length in DNS (RFC 1123)
const dns1123SubdomainMaxLength int = 253

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

// IsValidRFC1123 tests for a string that conforms to the definition of a
// subdomain in DNS (RFC 1123).
func IsValidRFC1123(value string) bool {
	if len(value) > dns1123SubdomainMaxLength {
		return false
	}
	if !dns1123SubdomainRegexp.MatchString(value) {
		return false
	}
	return true
}

var dns1123IllegalStartRegex = regexp.MustCompile(`^[^0-9a-z]+`)
var dns1123IllegalEndRegex = regexp.MustCompile(`[^0-9a-z]+$`)
var dns1123IllegalCharsRegex = regexp.MustCompile(`[^0-9a-z-.]`)

func ConvertToRFC1123(value string, args ...int) string {
	value = strings.ToLower(value)

	if len(value) == 0 || IsValidRFC1123(value) {
		return value
	}

	// failsafe
	depth := 1
	if len(args) > 0 {
		depth = args[0]
	}
	if depth == 50 {
		panic(fmt.Sprintf("failed to convert %q to valid dns 1123", value))
	}
	depth++

	value = dns1123IllegalStartRegex.ReplaceAllString(value, "")
	value = dns1123IllegalEndRegex.ReplaceAllString(value, "")
	value = dns1123IllegalCharsRegex.ReplaceAllString(value, "")

	if len(value) > dns1123SubdomainMaxLength {
		value = value[0:dns1123SubdomainMaxLength]
	}

	return ConvertToRFC1123(value, depth)
}

func getRoleLabelsImpl(config *embeddedclusterv1beta1.ConfigSpec, roles []string) []string {
	toReturn := []string{}

	if config == nil {
		return toReturn
	}

	for _, role := range roles {
		if role == config.Roles.Controller.Name {
			for k, v := range config.Roles.Controller.Labels {
				toReturn = append(toReturn, fmt.Sprintf("%s=%s", labelKey(k), labelify(v)))
			}
		}
		for _, customRole := range config.Roles.Custom {
			if role == customRole.Name {
				for k, v := range customRole.Labels {
					toReturn = append(toReturn, fmt.Sprintf("%s=%s", labelKey(k), labelify(v)))
				}
			}
		}
	}

	sort.Strings(toReturn)

	return toReturn
}
