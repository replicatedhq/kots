## IMPORTANT ##
# This file should only contain rules for linting RENDERED spec files
# Rego playground: https://play.openpolicyagent.org/

package kots.spec.rendered

# Files set with the contents of each file as json
files[output] {
  file := input[_]
  output := {
    "name": file.name,
    "path": file.path,
    "content": yaml.unmarshal(file.content),
    "docIndex": object.get(file, "docIndex", 0)
  }
}

# Check if any "status informer" has invalid format
is_informer_format_valid(informer) {
  is_string(informer)
  expression := "^(?:([^\/]+)\/)?([^\/]+)\/([^\/]+)$"
  matches := regex.find_all_string_submatch_n(expression, informer, -1)
  count(matches) > 0

  capture_groups := matches[0]
  count(capture_groups) == 4
} else {
  informer == ""
}
lint[output] {
  file := files[_]
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.kind == "Application"

  status_informers := file.content.spec.statusInformers
  count(status_informers) > 0

  informer := status_informers[i]
  not is_informer_format_valid(informer)

  field := sprintf("spec.statusInformers.%d", [i])
  output := {
    "rule": "invalid-status-informer-format",
    "type": "warn",
    "message": "Invalid status informer format",
    "path": file.path,
    "field": field,
    "docIndex": file.docIndex
  }
}

# Check if any "status informer" points to a non-existent object
informer_object_exists(informer) {
  is_string(informer)
  expression := "^(?:([^\/]+)\/)?([^\/]+)\/([^\/]+)$"
  matches := regex.find_all_string_submatch_n(expression, informer, -1)
  count(matches) > 0

  capture_groups := matches[0]
  count(capture_groups) == 4

  k8sObj := files[_].content
  is_string(k8sObj.kind)
  is_string(k8sObj.metadata.name)

  namespace := object.get(k8sObj.metadata, "namespace", "")
  type := lower(k8sObj.kind)
  name := k8sObj.metadata.name

  namespace == capture_groups[1]
  type == capture_groups[2]
  name == capture_groups[3]
} else {
  informer == ""
}
lint[output] {
  rule_name := "nonexistent-status-informer-object"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off

  file := files[_]
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.kind == "Application"

  status_informers := file.content.spec.statusInformers
  count(status_informers) > 0

  informer := status_informers[i]
  is_informer_format_valid(informer)
  not informer_object_exists(informer)

  field := sprintf("spec.statusInformers.%d", [i])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Status informer points to a nonexistent kubernetes object. If this is a Helm resource, this warning can be ignored.",
    "path": file.path,
    "field": field,
    "docIndex": file.docIndex
  }
}

# Check if LintConfig spec exists
lintconfig_spec_exists {
  file := files[_]
  file.content.kind == "LintConfig"
  file.content.apiVersion == "kots.io/v1beta1"
}

# Check if linting rule is ignored
lint_rule_config(lint_rule_name, default_level) = lint_rule_config {
  lintconfig_spec_exists
  lintconfig := files[_].content.spec
  lint_rule = lintconfig.rules[_]
  lint_rule.name == lint_rule_name
  rule_level := validate_lint_rule_level(default_level, lint_rule.level)
  lint_rule_config := {
    "off": lint_rule.level == "off",
    "level": rule_level
  }
} else = {
  "off": false,
  "level": default_level
}

# Validate linting rule level, use default if not valid
validate_lint_rule_level(default_level, input_level) = default_level {
  input_level != "error"
  input_level != "warn"
  input_level != "info"
  input_level != "off"
} else = input_level
