package kots.spec.builders

# Files set with the contents of each file as json
files[output] {
  file := input[_]
  output := {
    "name": file.name,
    "path": file.path,
    "content": yaml.unmarshal(file.content),
    "docIndex": object.get(file, "docIndex", 0),
    "allowDuplicates": object.get(file, "allowDuplicates", false)
  }
}

allowed_preflight_api_versions := {
  "troubleshoot.replicated.com/v1beta1",
  "troubleshoot.sh/v1beta2"
}

# Check if Preflight spec exists
preflight_spec_exists {
  file := files[_]
  file.content.kind == "Preflight"
  file.content.apiVersion == allowed_preflight_api_versions[_]
}

preflight_embedded_exists {
  kinds := {"Secret", "ConfigMap"}
  file := files[_]
  file.content.kind == kinds[_]
  file.content.apiVersion == "v1"
  file.content.metadata.labels["troubleshoot.sh/kind"] == "preflight"
  sd := file.content.stringData[_]
  string_data := yaml.unmarshal(sd)
  string_data.kind == "Preflight"
  string_data.apiVersion == allowed_preflight_api_versions[_]
}

lint[output] {
  rule_name := "preflight-spec"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  not preflight_spec_exists
  not preflight_embedded_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing preflight spec"
  }
}

# Check if informer labels exist
wanted_informer_labels = {
  "app.kubernetes.io/managed-by",
  "app.kubernetes.io/name",
  "app.kubernetes.io/instance"
}

informer_labels_present {
  file := files[_]

  all_defined_labels := { x | file.content.metadata.labels[x] }
  wanted_defined_labels := all_defined_labels & wanted_informer_labels
  wanted_defined_labels == wanted_informer_labels
}

lint[output] {
  rule_name := "informers-labels-not-found"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off

  not informer_labels_present

  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "No informer labels found on any resources"
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
