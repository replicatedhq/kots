package libyaml

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
	validator "gopkg.in/go-playground/validator.v8"
)

var (
	KeyRegExp             = regexp.MustCompile(`^([^\[]+)(?:\[(\d+)\])?$`)
	BytesRegExp           = regexp.MustCompile(`(?i)^(\d+(?:\.\d{1,3})?)([KMGTPE]B?)$`)
	K8sQuantityRegExp     = regexp.MustCompile(`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`)
	DockerVerLegacyRegExp = regexp.MustCompile(`^1\.([0-9]|(1[0-3]))\.[0-9]+$`)
	DockerVerRegExp       = regexp.MustCompile(`^[0-9]{2}\.((0[1-9])|(1[0-2]))\.[0-9]+(-(ce|ee))?$`)

	registeredValidationFuncs      = map[string]validator.Func{}
	registeredValidationErrorFuncs = map[string]ValidationErrorFunc{}
)

type ValidationErrorFunc func(formatted string, key string, fieldErr *validator.FieldError, root *RootConfig) error

func RegisterValidation(key string, validatorFn validator.Func, errorFn ValidationErrorFunc) {
	registeredValidationFuncs[key] = validatorFn
	registeredValidationErrorFuncs[key] = errorFn
}

// RegisterValidations will register all known validation for the libyaml project.
func RegisterValidations(v *validator.Validate) error {
	if err := v.RegisterValidation("int", IntValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("configitemtype", ConfigItemTypeValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("configitemwhen", ConfigItemWhenValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("apiversion", SemverValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("dockerversion", DockerVersionValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("semver", SemverValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("semverrange", SemverRangeValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("hastarget", HasTargetValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("componentexists", ComponentExistsValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("containerexists", ContainerExistsValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("componentcontainer", ComponentContainerFormatValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("clusterstrategy", ClusterStrategyValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("absolutepath", IsAbsolutePathValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("volumeoptions", VolumeOptionsValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("isempty", IsEmptyValidation); err != nil {
		return err
	}

	// will handle this in vendor web. this prevents panic from validator.v8 library
	if err := v.RegisterValidation("integrationexists", NoopValidation); err != nil {
		return err
	}

	// will handle this in vendor web. this prevents panic from validator.v8 library
	if err := v.RegisterValidation("externalregistryexists", NoopValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("bytes", IsBytesValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("quantity", IsK8sQuantityValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("bool", IsBoolValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("uint", IsUintValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("tcpport", IsTCPUDPPortValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("udpport", IsTCPUDPPortValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("graphiteretention", GraphiteRetentionFormatValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("graphiteaggregation", GraphiteAggregationFormatValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("monitorlabelscale", MonitorLabelScaleValidation); err != nil {
		return err
	}

	if err := v.RegisterValidation("fingerprint", Fingerprint); err != nil {
		return err
	}

	if err := v.RegisterValidation("shellalias", ShellAlias); err != nil {
		return err
	}

	if err := v.RegisterValidation("url", URLValid); err != nil {
		return err
	}

	if err := v.RegisterValidation("containernameexists", ContainerNameExists); err != nil {
		return err
	}

	if err := v.RegisterValidation("containernameunique", ContainerNameUnique); err != nil {
		return err
	}

	if err := v.RegisterValidation("clusterinstancefalse", ClusterInstanceFalse); err != nil {
		return err
	}

	if err := v.RegisterValidation("requiressubscription", RequiresSubscription); err != nil {
		return err
	}

	if err := v.RegisterValidation("customrequirementidunique", CustomRequirementIDUnique); err != nil {
		return err
	}

	if err := v.RegisterValidation("mapkeylengthnonzero", MapKeyLengthNonZero); err != nil {
		return err
	}

	if err := v.RegisterValidation("required_minapiversion", RequiredMinAPIVersion); err != nil {
		return err
	}

	for key, fn := range registeredValidationFuncs {
		if err := v.RegisterValidation(key, fn); err != nil {
			return err
		}
	}

	return nil
}

func FormatFieldError(key string, fieldErr *validator.FieldError, root *RootConfig) error {
	formatted, err := FormatKey(key, fieldErr, root)
	if err != nil {
		formatted = key
	}

	switch fieldErr.Tag {
	case "apiversion":
		return fmt.Errorf("A valid \"replicated_api_version\" is required as a root element")

	case "dockerversion":
		return fmt.Errorf("Invalid Docker version suppiled in %q", formatted)

	case "semver":
		return fmt.Errorf("Invalid version suppiled in %q", formatted)

	case "semverrange":
		return fmt.Errorf("Invalid version range suppiled in %q", formatted)

	case "hastarget":
		return fmt.Errorf("Custom monitor is missing targets at key %q", formatted)

	case "componentexists":
		return fmt.Errorf("Component %q does not exist at key %q", fieldErr.Value, formatted)

	case "containerexists":
		return fmt.Errorf("Container %q does not exist at key %q", fieldErr.Value, formatted)

	case "componentcontainer":
		return fmt.Errorf("Should be in the format \"<component name>,<container image name>\" at key %q", formatted)

	case "clusterstrategy":
		return fmt.Errorf("Invalid strategy value at key %q. Valid values are \"autoscale\", \"random\".", formatted)

	case "volumeoptions":
		return fmt.Errorf("Invalid volume option list %q", formatted)

	case "integrationexists":
		return fmt.Errorf("Missing integration %q at key %q", fieldErr.Value, formatted)

	case "externalregistryexists":
		return fmt.Errorf("Missing external registry integration %q at key %q", fieldErr.Value, formatted)

	case "bytes":
		return fmt.Errorf("Byte quantity key %q must be a positive decimal with a unit of measurement like M, MB, G, or GB", formatted)

	case "quantity", "bytes|quantity":
		return fmt.Errorf("Quantity at key %q must be expressed as a plain integer, a fixed-point integer, or the power-of-two equivalent (e.g. 128974848, 129e6, 129M, 123Mi)", formatted)

	case "required":
		return fmt.Errorf("Value required at key %q", formatted)

	case "tcpport":
		return fmt.Errorf("A valid port number must be between 0 and 65535: %q", formatted)

	case "graphiteretention":
		return fmt.Errorf("Should be in the new style graphite retention policy at key %q", formatted)

	case "graphiteaggregation":
		return fmt.Errorf("Valid values for graphite aggregation method are 'average', 'sum', 'min', 'max', 'last' at key %q", formatted)

	case "monitorlabelscale":
		return fmt.Errorf("Please specify 'metric', 'none', or a floating point number for scale at %q", formatted)

	case "shellalias":
		return fmt.Errorf("Valid characters for shell alias are [a-zA-Z0-9_\\-] at %q", formatted)

	case "url":
		return fmt.Errorf("A valid URL accessible from the internet is required at %q", formatted)

	case "fingerprint":
		return fmt.Errorf("Please specify a valid RFC4716 key fingerprint at %q", formatted)

	case "containernameexists":
		return fmt.Errorf("Container name %q does not exist at key %q", fieldErr.Value, formatted)

	case "containernameunique":
		return fmt.Errorf("Container name %q is required to be unique at key %q", fieldErr.Value, formatted)

	case "clusterinstancefalse":
		return fmt.Errorf("Cluster must be set to false for container at key %q", formatted)

	case "requiressubscription":
		return fmt.Errorf("Failed to traverse subscription tree from key %q to container with name %q", formatted, fieldErr.Value)

	case "customrequirementidunique":
		return fmt.Errorf("Custom requirement %q is required to be unique at key %q", fieldErr.Value, formatted)

	case "mapkeylengthnonzero":
		return fmt.Errorf("Map keys are required to have a length greater than zero: %q", formatted)

	case "required_minapiversion":
		return fmt.Errorf("Field is required for \"min_api_version\" < %s at key %q", fieldErr.Param, formatted)

	default:
		if fn, ok := registeredValidationErrorFuncs[fieldErr.Tag]; ok {
			return fn(formatted, key, fieldErr, root)
		}

		return fmt.Errorf("Validation failed on the %q tag at key %q", fieldErr.Tag, formatted)
	}
}

func FormatKey(keyChain string, fieldErr *validator.FieldError, root *RootConfig) (string, error) {
	value := reflect.ValueOf(*root)
	keys := strings.Split(keyChain, ".")

	rest, err := formatKey(keys, value)
	if err != nil {
		return "", err
	}

	if rest != "" {
		rest = rest[1:]

		matches := KeyRegExp.FindStringSubmatch(fieldErr.Field)
		if matches[2] != "" {
			rest += fmt.Sprintf("[%s]", matches[2])
		}
	}

	return rest, nil
}

func formatKey(keys []string, parent reflect.Value) (string, error) {
	if len(keys) == 1 {
		return "", nil
	}

	if parent.Type().Kind() == reflect.Ptr {
		parent = parent.Elem()
	}

	if parent.Type().Kind() == reflect.Struct {
		key := keys[1]
		matches := KeyRegExp.FindStringSubmatch(key)

		field, ok := parent.Type().FieldByName(matches[1])
		if !ok {
			return "", fmt.Errorf("field %q not found", matches[1])
		}

		yamlTag := field.Tag.Get("yaml")
		yamlTagParts := strings.Split(yamlTag, ",")
		if len(yamlTagParts) > 0 {
			yamlTag = yamlTagParts[0]
		}

		value := parent.FieldByName(matches[1])

		rest, err := formatKey(keys[1:], value)
		if err != nil {
			return "", err
		}

		return "." + yamlTag + rest, nil
	} else if parent.Type().Kind() == reflect.Slice {
		key := keys[0]
		matches := KeyRegExp.FindStringSubmatch(key)

		i, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", err
		}

		value := parent.Index(i)

		rest, err := formatKey(keys, value)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%d]", i) + rest, nil
	}

	return "", nil
}

var intRegex = regexp.MustCompile(`^[-+]?[0-9]+$`)

// IntValidation is the validation function for validating if the current field's value is a valid integer
func IntValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return intRegex.MatchString(field.String())
}

// ConfigItemTypeValidation will validate that the type element of a config item is a supported and valid option.
func ConfigItemTypeValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return false
	}

	validTypes := map[string]bool{
		"text":        true,
		"label":       true,
		"password":    true,
		"file":        true,
		"bool":        true,
		"select_one":  true,
		"select_many": true,
		"textarea":    true,
		"select":      true,
		"heading":     true,
	}

	if validTypes[field.String()] {
		return true
	}

	return false
}

// ConfigItemWhenValidation will validate that the when element of a config item is in a valid format and references other valid, created objects.
func ConfigItemWhenValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	var whenValue string

	whenValue = field.String()
	if whenValue == "" {
		return true
	}

	// new style
	if hasReplTemplate(field) {
		return true
	}
	if _, err := strconv.ParseBool(whenValue); err == nil {
		return true
	}

	splitString := "="
	if strings.Contains(whenValue, "!=") {
		splitString = "!="
	}

	parts := strings.SplitN(whenValue, splitString, 2)
	if len(parts) >= 2 {
		whenValue = parts[0]
	}

	return configItemExists(whenValue, root)
}

func configItemExists(configItemName string, root *RootConfig) bool {
	for _, group := range root.ConfigGroups {
		for _, item := range group.Items {
			if item != nil && item.Name == configItemName {
				return true
			}
			if item != nil {
				for _, childItem := range item.Items {
					if childItem != nil && childItem.Name == configItemName {
						return true
					}
				}
			}
		}
	}

	return false
}

func componentExists(componentName string, root *RootConfig) bool {
	for _, component := range root.Components {
		if component != nil && component.Name == componentName {
			return true
		}
	}

	return false
}

func containerExists(componentName, containerName string, root *RootConfig) bool {
	for _, component := range root.Components {
		if component != nil && component.Name == componentName {
			for _, container := range component.Containers {
				if container != nil && container.ImageName == containerName {
					return true
				}
			}
			return false
		}
	}

	return false
}

// HasTargetValidation validates that all custom monitors have at least one target defined
func HasTargetValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	customMonitors, ok := field.Interface().([]CustomMonitor)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	for _, monitor := range customMonitors {
		if len(monitor.Target) == 0 && len(monitor.Targets) == 0 {
			return false
		}
	}
	return true
}

// ComponentExistsValidation will validate that the specified component name is present in the current YAML.
func ComponentExistsValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	// validates that the component exists in the root.Components slice
	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	var componentName string

	componentName = field.String()

	parts := strings.SplitN(componentName, ",", 2)

	if len(parts) == 1 {
		//This might be a swarm config
		//If the scheduler is swarm, accept it
		if IsSwarm(root) {
			return true
		}
	}

	if len(parts) >= 2 {
		componentName = parts[0]
	}

	return componentExists(componentName, root)
}

// ContainerExistsValidation will validate that the specified container name is present in the current YAML.
func ContainerExistsValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	// validates that the container exists in the root.components.containers slice

	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	var componentName, containerName string

	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	containerName = field.String()

	if param != "" {
		componentField, componentKind, ok := v.GetStructFieldOK(currentStructOrField, param)

		if !ok || componentKind != reflect.String {
			// this is an issue with the code and really should be a panic
			return true
		}

		componentName = componentField.String()
	} else {
		parts := strings.SplitN(containerName, ",", 2)

		if len(parts) < 2 {
			// let "componentcontainer" validation handle this case
			return true
		}

		componentName = parts[0]
		containerName = parts[1]
	}

	if !componentExists(componentName, root) {
		// let "componentexists" validation handle this case
		return true
	}

	return containerExists(componentName, containerName, root)
}

// IsAbsolutePathValidation validates that the format of the field begins with a "/" unless is't a repl template
func IsAbsolutePathValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	return strings.HasPrefix(field.String(), "/") || strings.HasPrefix(field.String(), "{{repl ")
}

// VolumeOptionsValidation checks that volume option list does not contain any conlicting or duplicate options.
func VolumeOptionsValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	// TODO: There are more rules.  Look for errInvalidMode in docker source code.
	// Specifically, the nocopy option requires some additional checks.

	if fieldKind != reflect.Slice {
		// this is an issue with the code and really should be a panic
		return true
	}

	options, ok := field.Interface().([]string)
	if !ok {
		return false
	}

	// Only one of each is allowed.
	rwModes := map[string]bool{"rw": true, "ro": true}
	labelModes := map[string]bool{"z": true, "Z": true}
	propModes := map[string]bool{"shared": true, "slave": true, "private": true, "rshared": true, "rslave": true, "rprivate": true}
	copyModes := map[string]bool{"nocopy": true}

	numRwModes := 0
	numLabelModes := 0
	numPropModes := 0
	numCopyModes := 0
	for _, o := range options {
		switch {
		case rwModes[o]:
			numRwModes++
		case labelModes[o]:
			numLabelModes++
		case propModes[o]:
			numPropModes++
		case copyModes[o]:
			numCopyModes++
		default:
			return false
		}
	}

	return numRwModes < 2 && numLabelModes < 2 && numPropModes < 2 && numCopyModes < 2
}

// ComponentContainerFormatValidation will validate that component/container name is in the correct format.
func ComponentContainerFormatValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	// validates the format of the string field conforms to "<component name>,<container image name>"

	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	parts := strings.SplitN(field.String(), ",", 2)

	if len(parts) == 1 {
		//This might be a swarm config
		//If the scheduler is swarm, accept it
		root, ok := topStruct.Interface().(*RootConfig)
		if !ok {
			// this is an issue with the code and really should be a panic
			return true
		}
		return IsSwarm(root)
	}

	if len(parts) < 2 {
		return false
	}

	return true
}

// ClusterStrategyValidation will validate that component/container name is in the correct format.
func ClusterStrategyValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	return field.String() == "autoscale" || field.String() == "random"
}

// DockerVersionValidation will validate that the field is in correct, proper docker version format.
func DockerVersionValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	// new style docker version e.g. 17.03.0-ce
	if DockerVerRegExp.MatchString(field.String()) {
		return true
	}

	// legacy style docker version e.g. 1.13.1
	if DockerVerLegacyRegExp.MatchString(field.String()) {
		return true
	}

	return false
}

// SemverValidation will validate that the field is in correct, proper semver format.
func SemverValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	_, err := semver.Make(field.String())
	return err == nil
}

// SemverRangeValidation will validate that the field is in correct, proper semver format.
func SemverRangeValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	_, err := semver.ParseRange(field.String())
	return err == nil
}

// IsBytesValidation will return if a field is a parseable bytes value.
func IsBytesValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	parts := BytesRegExp.FindStringSubmatch(strings.TrimSpace(field.String()))
	if len(parts) < 3 {
		return false
	}

	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || value <= 0 {
		return false
	}

	return true
}

// IsK8sQuantityValidation will return if a field is a parseable kubernetes resource.Quantity.
// https://github.com/kubernetes/apimachinery/blob/2de00c78cb6d6127fb51b9531c1b3def1cbcac8c/pkg/api/resource/quantity.go#L144
func IsK8sQuantityValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	return K8sQuantityRegExp.MatchString(strings.TrimSpace(field.String()))
}

// IsBoolValidation will return if a string field parses to a bool.
func IsBoolValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	_, err := strconv.ParseBool(field.String())
	if err != nil {
		return false
	}

	return true
}

// IsUintValidation will return if a string field parses to a uint.
func IsUintValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	_, err := strconv.ParseUint(field.String(), 10, 64)
	if err != nil {
		return false
	}

	return true
}

// IsTCPUDPPortValidation will return true if a field value is also a valid TCP port.
func IsTCPUDPPortValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.Int32 {
		// this is an issue with the code and really should be a panic
		return true
	}

	port := field.Int()
	return 0 <= port && port <= 65535
}

func IsEmptyValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	return false
}

// NoopValidation will return true always.
func NoopValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return true
}

// GraphiteRetentionFormatValidation will return true if the field value is a valid graphite retention value.
func GraphiteRetentionFormatValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	validate := func(amount, unit string) bool {
		_, err := strconv.Atoi(amount)
		if err != nil {
			return false
		}

		switch unit {
		case "s", "m", "h", "d", "y":
			return true
		default:
			return false
		}
	}

	// Example: 15s:7d,1m:21d,15m:5y
	periods := strings.Split(valueStr, ",")
	for _, period := range periods {
		periodParts := strings.Split(period, ":")
		if len(periodParts) != 2 {
			return false
		}

		for _, part := range periodParts {
			partLen := len(part)
			if partLen < 2 {
				return false
			}
			if !validate(part[:partLen-1], part[partLen-1:]) {
				return false
			}
		}
	}

	return true
}

// GraphiteAggregationFormatValidation will return true if the field value is a valid value for the a graphite aggregation.
func GraphiteAggregationFormatValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	switch valueStr {
	case "average", "sum", "min", "max", "last":
		return true
	default:
		return false
	}
}

// MonitorLabelScaleValidation will return true only if the value is a parseable and correct value for the scale.
func MonitorLabelScaleValidation(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	switch valueStr {
	case "metric", "none":
		return true
	default:
		_, err := strconv.ParseFloat(valueStr, 64)
		return err == nil
	}
}

// Validates MD5 fingerprint defined in RFC4716
func Fingerprint(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	// Valid fingerprints look like this: cb:69:19:cd:76:1f:17:54:92:a4:fc:a9:6f:a5:57:72
	octets := strings.Split(valueStr, ":")
	if len(octets) != 16 {
		return false
	}
	for _, o := range octets {
		valid, err := regexp.MatchString("[a-f0-9][a-f0-9]", o)
		if err != nil || !valid {
			return false
		}
	}

	return true
}

// ShellAlias validates that the string is a suitable to be used as a shell alias
func ShellAlias(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	valid, err := regexp.MatchString("^[a-zA-Z0-9_\\-]*$", valueStr)
	if err != nil || !valid {
		return false
	}

	return true
}

func URLValid(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	imageUrl := field.String()
	if imageUrl == "" {
		return true
	}

	parsed, err := url.Parse(imageUrl)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	if parsed.Host == "" {
		return false
	}

	return true
}

func ContainerNameExists(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	containerName := field.String()
	if containerName == "" {
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	currentContainer := getCurrentContainer(currentStructOrField)
	if currentContainer == nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	container := getContainerFromName(containerName, currentContainer, root)
	if container == nil {
		return false
	}
	return true
}

func ContainerNameUnique(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	containerName := field.String()
	if containerName == "" {
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	currentContainer := getCurrentContainer(currentStructOrField)
	if currentContainer == nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	container := getContainerFromName(containerName, currentContainer, root)
	if container != nil {
		return false
	}
	return true
}

func ClusterInstanceFalse(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	if valueStr == "" {
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	currentContainer := getCurrentContainer(currentStructOrField)
	if currentContainer == nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	cluster, err := currentContainer.Cluster.Parse()
	if err != nil {
		// don't worry about this here. cluster should have the "bool" validator.
		return true
	}

	if cluster {
		return false
	}

	return true
}

func MapKeyLengthNonZero(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.Map {
		return true
	}

	for _, k := range field.MapKeys() {
		if len(k.String()) == 0 {
			return false
		}
	}

	return true
}

func RequiresSubscription(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	containerName := field.String()
	if containerName == "" {
		return true
	}

	if hasReplTemplate(field) {
		// all bets are off
		return true
	}

	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	currentContainer := getCurrentContainer(currentStructOrField)
	if currentContainer == nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	currentComponent := getCurrentComponentFromContainer(currentContainer, root)
	if currentComponent == nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	subscribedContainer := getContainerFromName(containerName, currentContainer, root)
	if subscribedContainer == nil {
		return false
	}

	subscribedComponent := getCurrentComponentFromContainer(subscribedContainer, root)
	if subscribedComponent == nil {
		return false
	}

	subscriptions := buildSubscriptionMap(root)

	current := fmt.Sprintf("%s:%s", currentComponent.Name, currentContainer.ImageName)
	subscribed := fmt.Sprintf("%s:%s", subscribedComponent.Name, subscribedContainer.ImageName)
	if !dependsOn(subscriptions, current, subscribed) {
		return false
	}

	return true
}

func RequiredMinAPIVersion(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	root, ok := topStruct.Interface().(APIVersioner)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	apiVer, err := semver.Make(root.GetAPIVersion())
	if err != nil {
		// this will get caught in min_api_version field validation
		return true
	}

	minVer, err := semver.Make(param)
	if err != nil {
		// this is an issue with the code and really should be a panic
		return true
	}

	if apiVer.GTE(minVer) {
		return true
	}

	return validator.HasValue(v, topStruct, currentStructOrField, field, fieldType, fieldKind, "")
}

func hasReplTemplate(field reflect.Value) bool {
	return strings.Contains(field.String(), "{{repl ")
}

func getCurrentContainer(current reflect.Value) *Container {
	if !current.CanAddr() {
		return nil
	}
	currentContainer, _ := current.Addr().Interface().(*Container)
	return currentContainer
}

func getCurrentComponentFromContainer(current *Container, root *RootConfig) *Component {
	for _, component := range root.Components {
		for _, container := range component.Containers {
			if current == container {
				return component
			}
		}
	}
	return nil
}

func getContainerFromName(name string, current *Container, root *RootConfig) *Container {
	for _, component := range root.Components {
		for _, container := range component.Containers {
			if current == container {
				continue
			}
			if container.Name == name {
				return container
			}
		}
	}
	return nil
}

func buildSubscriptionMap(root *RootConfig) map[string]string {
	result := make(map[string]string)
	for _, component := range root.Components {
		for _, container := range component.Containers {
			for _, p := range container.PublishEvents {
				for _, s := range p.Subscriptions {
					if s.Action != "start" {
						continue
					}
					result[fmt.Sprintf("%s:%s", s.ComponentName, s.ContainerName)] = fmt.Sprintf("%s:%s", component.Name, container.ImageName)
				}
			}
		}
	}
	return result
}

func dependsOn(subscriptions map[string]string, current string, subscribed string) bool {
	nextCurrent, ok := subscriptions[current]
	if !ok {
		return false
	}
	if nextCurrent == subscribed {
		return true
	}
	nextSubscriptions := make(map[string]string)
	for k, v := range subscriptions {
		if k != current {
			nextSubscriptions[k] = v
		}
	}
	return dependsOn(nextSubscriptions, nextCurrent, subscribed)
}

func IsSwarm(root *RootConfig) bool {
	return root.Swarm != nil
}
