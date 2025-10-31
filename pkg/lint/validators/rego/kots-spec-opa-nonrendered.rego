## IMPORTANT ##
# This file should only contain rules for linting NON-rendered spec files
# Rego playground: https://play.openpolicyagent.org/

package kots.spec.nonrendered

## Secrets with template functions are excluded in the rule logic
secrets_regular_expressions = [
  # connection strings with username and password
  # http://user:password@host:8888
  "(?i)(https?|ftp)(:\\/\\/)[^:\"\\/]+(:)[^@\"\/]+@[^:\\/\\s\"]+:[\\d]+",
  # user:password@tcp(host:3309)/db-name
  "\\b[^:\"\\/]*(:)[^:\"\\/]*(@tcp\\()[^:\"\\/]*:[\\d]*?(\\)\\/)[\\w\\d\\S-_]+\\b",
  # passwords & tokens (stringified jsons)
  "(?i)(\\\"name\\\":\\\"[^\"]*password[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*token[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*database[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*user[^\"]*\\\",\\\"value\\\":\\\")",
  # passwords & tokens (in YAMLs)
  "(?i)(name: [\"']{0,1}password[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}token[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}database[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}user[\"']{0,1})\n\\s*(value:)",
  "(?i)password: .*",
  "(?i)token: .*",
  "(?i)database: .*",
  "(?i)user: .*",
  # standard postgres and mysql connnection strings
  "(?i)(Data Source *= *)[^\\;]+(;)",
  "(?i)(location *= *)[^\\;]+(;)",
  "(?i)(User ID *= *)[^\\;]+(;)",
  "(?i)(password *= *)[^\\;]+(;)",
  "(?i)(Server *= *)[^\\;]+(;)",
  "(?i)(Database *= *)[^\\;]+(;)",
  "(?i)(Uid *= *)[^\\;]+(;)",
  "(?i)(Pwd *= *)[^\\;]+(;)",
  # AWS secrets
  "SECRET_?ACCESS_?KEY",
  "ACCESS_?KEY_?ID",
  "OWNER_?ACCOUNT",
]

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

# Returns the string value of x
string(x) = y {
	y := split(yaml.marshal(x), "\n")[0]
}

# A set containing ALL the specs for each file
# 3 levels deep. "specs" rule for each level
specs[output] {
  file := files[_]
  spec := file.content.spec # 1st level
  output := {
    "path": file.path,
    "spec": spec,
    "field": "spec",
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key].spec # 2nd level
  field := concat(".", [string(key), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key1][key2].spec # 3rd level
  field := concat(".", [string(key1), string(key2), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}

# A set containing all of the kots kinds
is_troubleshoot_api_version(apiVersion) {
  apiVersion == "troubleshoot.replicated.com/v1beta1"
} else {
  apiVersion == "troubleshoot.sh/v1beta2"
}
is_kubernetes_installer_api_version(apiVersion) {
  apiVersion == "cluster.kurl.sh/v1beta1"
} else {
  apiVersion == "kurl.sh/v1beta1"
}
is_kots_kind(file) {
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.kind == "Config"
} else {
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.kind == "Application"
} else {
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.kind == "Identity"
} else {
  is_troubleshoot_api_version(file.content.apiVersion)
  file.content.kind == "Collector"
} else {
  is_troubleshoot_api_version(file.content.apiVersion)
  file.content.kind == "Analyzer"
} else {
  is_troubleshoot_api_version(file.content.apiVersion)
  file.content.kind == "SupportBundle"
} else {
  is_troubleshoot_api_version(file.content.apiVersion)
  file.content.kind == "Redactor"
} else {
  is_troubleshoot_api_version(file.content.apiVersion)
  file.content.kind == "Preflight"
} else {
  file.content.apiVersion == "velero.io/v1"
  file.content.kind == "Backup"
} else {
  file.content.apiVersion == "velero.io/v1"
  file.content.kind == "Restore"
} else {
  is_kubernetes_installer_api_version(file.content.apiVersion)
  file.content.kind == "Installer"
} else {
  file.content.apiVersion == "app.k8s.io/v1beta1"
  file.content.kind == "Application"
} else {
  file.content.apiVersion == "app.k8s.io/v1beta1"
  file.content.kind == "LintConfig"
}
kots_kinds[output] {
  file := files[_]
  is_kots_kind(file)
  output := {
    "apiVersion": file.content.apiVersion,
    "kind": file.content.kind,
    "filePath": file.path,
    "docIndex": file.docIndex,
    "allowDuplicates": file.allowDuplicates
  }
}

# A rule that returns the config file path
config_file_path = file.path {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
}

# A rule that returns the config data
config_data = output {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
  output := {
    "config": file.content.spec,
    "field": "spec",
    "docIndex": file.docIndex
  }
}

# A set containing all of the config groups, config items and child items
# Config Groups
config_options[output] {
  item := config_data.config.groups[index]
  field := concat(".", [config_data.field, "groups", string(index)])
  output := {
    "item": item,
    "field": field
  }
}
# Config Items
config_options[output] {
  item := config_data.config.groups[groupIndex].items[itemIndex]
  field := concat(".", [config_data.field, "groups", string(groupIndex), "items", string(itemIndex)])
  output := {
    "item": item,
    "field": field
  }
}
# Config Child Items
config_options[output] {
  item := config_data.config.groups[groupIndex].items[itemIndex].items[childItemIndex]
  field := concat(".", [config_data.field, "groups", string(groupIndex), "items", string(itemIndex), "items", string(childItemIndex)])
  output := {
    "item": item,
    "field": field
  }
}

# A function that checks if a config option exists in config
config_option_exists(option_name) {
  option := config_options[_].item
  option.name == option_name
}

# A function that checks if a config option is repeatable
config_option_is_repeatable(option_name) {
  option := config_options[_].item
  option.name == option_name
  option.repeatable
}

template_yamlPath_ends_with_array(template) {
  not template.yamlPath == ""
  expression := "(.*)\\[[0-9]\\]$"
  re_match(expression, template.yamlPath)
}

# Check if any files are missing "kind"
lint[output] {
  rule_name := "missing-kind-field"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  file := files[_]
  not file.content.kind
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing \"kind\" field",
    "path": file.path,
    "docIndex": file.docIndex
  }
}

# Check if any files are missing "apiVersion"
lint[output] {
  rule_name := "missing-api-version-field"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  file := files[_]
  not file.content.apiVersion
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing \"apiVersion\" field",
    "path": file.path,
    "docIndex": file.docIndex
  }
}

# Check if Preflight spec exists
v1beta1_preflight_spec_exists {
  file := files[_]
  file.content.kind == "Preflight"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_preflight_spec_exists {
  file := files[_]
  file.content.kind == "Preflight"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
lint[output] {
  rule_name := "preflight-spec"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  not v1beta1_preflight_spec_exists
  not v1beta2_preflight_spec_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing preflight spec"
  }
}

# Check if Config spec exists
config_spec_exists {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
}
lint[output] {
  rule_name := "config-spec"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  not config_spec_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing config spec"
  }
}

# Check if Troubleshoot spec exists
v1beta1_troubleshoot_spec_exists {
  file := files[_]
  file.content.kind == "Collector"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_troubleshoot_spec_exists {
  file := files[_]
  file.content.kind == "Collector"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
v1beta1_supportbundle_spec_exists {
  file := files[_]
  file.content.kind == "SupportBundle"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_supportbundle_spec_exists {
  file := files[_]
  file.content.kind == "SupportBundle"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
lint[output] {
  rule_name := "troubleshoot-spec"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  not v1beta1_troubleshoot_spec_exists
  not v1beta2_troubleshoot_spec_exists
  not v1beta1_supportbundle_spec_exists
  not v1beta2_supportbundle_spec_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing troubleshoot spec"
  }
}

# Check if Application spec exists
application_spec_exists {
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
}
lint[output] {
  rule_name := "application-spec"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  not application_spec_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing application spec"
  }
}

# Check if Application icon exists
lint[output] {
  rule_name := "application-icon"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
  not file.content.spec.icon
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing application icon",
    "path": file.path,
    "field": "spec",
    "docIndex": file.docIndex
  }
}

lint[output] {
  rule_name := "application-statusInformers"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
  not file.content.spec.statusInformers
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing application statusInformers",
    "path": file.path,
    "field": "spec",
    "docIndex": file.docIndex
  }
}

# Check if targetKotsVersion in the Application spec is a valid semver
lint[output] {
  rule_name := "invalid-target-kots-version"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.spec.targetKotsVersion
  not semver.is_valid(trim_prefix(file.content.spec.targetKotsVersion, "v"))
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Target KOTS version must be a valid semver",
    "path": file.path,
    "field": "spec.targetKotsVersion",
    "docIndex": file.docIndex
  }
}

# Check if minKotsVersion in the Application spec is a valid semver
lint[output] {
  rule_name := "invalid-min-kots-version"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
  file.content.spec.minKotsVersion
  not semver.is_valid(trim_prefix(file.content.spec.minKotsVersion, "v"))
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Minimum KOTS version must be a valid semver",
    "path": file.path,
    "field": "spec.minKotsVersion",
    "docIndex": file.docIndex
  }
}

# Check if helm charts in the embedded cluster config contain a version
lint[output] {
  rule_name := "ec-helm-extension-version-required"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  spec := specs[_]
  chart := spec.spec.extensions.helm.charts[index]
  not chart.version
  field := concat(".", [spec.field, "extensions.helm.charts", string(index)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing version for Helm Chart extension",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if the kubernetes installer addons versions are valid
is_kubernetes_installer(file) {
  is_kubernetes_installer_api_version(file.content.apiVersion)
  file.content.kind == "Installer"
}
is_addon_version_invalid(version) {
  contains(version, ".x")
} else {
  version == "latest"
}
lint[output] {
  rule_name := "invalid-kubernetes-installer"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  file := files[_]
  is_kubernetes_installer(file)
  is_addon_version_invalid(file.content.spec[addon].version)
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Add-ons included in the Kubernetes installer must pin specific versions rather than 'latest' or x-ranges (e.g., 1.2.x).",
    "path": file.path,
    "field": sprintf("spec.%s.version", [string(addon)]),
    "docIndex": file.docIndex
  }
}

# Check if the kubernetes installer is using the old deprecated api version
lint[output] {
  rule_name := "deprecated-kubernetes-installer-version"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  file := files[_]
  file.content.kind == "Installer"
  file.content.apiVersion == "kurl.sh/v1beta1"
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "API version 'kurl.sh/v1beta1' is deprecated. Use 'cluster.kurl.sh/v1beta1' instead.",
    "path": file.path,
    "field": "apiVersion",
    "docIndex": file.docIndex
  }
}

# Check if there are any duplicate kots kinds included
is_same_kots_kind(k1, k2) {
  k1.apiVersion == k2.apiVersion
  k1.kind == k2.kind
} else {
  is_troubleshoot_api_version(k1.apiVersion)
  is_troubleshoot_api_version(k2.apiVersion)
  k1.kind == k2.kind
} else {
  is_kubernetes_installer_api_version(k1.apiVersion)
  is_kubernetes_installer_api_version(k2.apiVersion)
  k1.kind == k2.kind
}
lint[output] {
  rule_name := "duplicate-kots-kind"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  kki := kots_kinds[i]
  kkj := kots_kinds[j]
  i != j
  not kki.allowDuplicates
  not kkj.allowDuplicates
  is_same_kots_kind(kki, kkj)
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": sprintf("A release can only include one '%s' resource, but another '%s' resource was found in %s", [string(kki.kind), string(kki.kind), string(kkj.filePath)]),
    "path": kki.filePath,
    "field": "apiVersion",
    "docIndex": kki.docIndex
  }
}

helm_chart_release_name_from_file(file) = releaseName {
  file.content.apiVersion == "kots.io/v1beta1"
  releaseName = {
    "value": file.content.spec.chart.releaseName,
    "field": "spec.chart.releaseName"
  }
} else = releaseName {
  file.content.apiVersion == "kots.io/v1beta2"
  releaseName = {
    "value": file.content.spec.releaseName,
    "field": "spec.releaseName"
  }
}

# A set containing all of the release names included in HelmChart CRDs
helm_release_names[output] {
  file := files[_]
  file.content.kind == "HelmChart"
  releaseName := helm_chart_release_name_from_file(file)
  output := {
    "filePath": file.path,
    "docIndex": file.docIndex,
    "apiVersion": file.content.apiVersion,
    "releaseName": releaseName.value,
    "field": releaseName.field
  }
}

# Check if the releaseName field in HelmChart CRDs is valid
is_valid_helm_release_name(name) {
  regex.match("^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$", name)
  count(name) <= 53
}
lint[output] {
  rule_name := "invalid-helm-release-name"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  rn := helm_release_names[_]
  not is_valid_helm_release_name(rn.releaseName)
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Invalid Helm release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
    "path": rn.filePath,
    "field": rn.field,
    "docIndex": rn.docIndex
  }
}

# Check if the releaseName field in HelmChart CRDs is unique across all charts
lint[output] {
  rule_name := "duplicate-helm-release-name"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  rni := helm_release_names[i]
  rnj := helm_release_names[j]
  i != j
  rni.apiVersion == rnj.apiVersion
  rni.releaseName == rnj.releaseName
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": sprintf("Release name is already used in %s", [string(rnj.filePath)]),
    "path": rni.filePath,
    "field": rni.field,
    "docIndex": rni.docIndex
  }
}

# Check if any spec has "replicas" set to 1
lint[output] {
  rule_name := "replicas-1"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  spec.spec.replicas == 1
  field := concat(".", [spec.field, "replicas"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Found Replicas 1",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any spec has "privileged" set to true
lint[output] {
  rule_name := "privileged"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  spec.spec.privileged == true
  field := concat(".", [spec.field, "privileged"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Found privileged spec",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any spec has "allowPrivilegeEscalation" set to true
lint[output] {
  rule_name := "allow-privilege-escalation"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  spec.spec.allowPrivilegeEscalation == true
  field := concat(".", [spec.field, "allowPrivilegeEscalation"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Allows privilege escalation",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container Image" contains the tag ":latest"
lint[output] {
  rule_name := "container-image-latest-tag"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  is_string(container.image)
  endswith(container.image, ":latest")
  field := concat(".", [spec.field, "containers", string(index), "image"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Container has image with tag 'latest'",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container Image" uses "LocalImageName"
lint[output] {
  rule_name := "container-image-local-image-name"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  is_string(container.image)
  re_match("^(repl{{|{{repl)\\s*LocalImageName", container.image)
  field := concat(".", [spec.field, "containers", string(index), "image"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Container image utilizes LocalImageName",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container" of a spec doesn’t have field "resources"
lint[output] {
  rule_name := "container-resources"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  not container.resources
  field := concat(".", [spec.field, "containers", string(index)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing container resources",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource" doesn’t have field "limits"
lint[output] {
  rule_name := "container-resource-limits"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources
  not container.resources.limits
  field := concat(".", [spec.field, "containers", string(index), "resources"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing resource limits",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource" doesn’t have field "requests"
lint[output] {
  rule_name := "container-resource-requests"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources
  not container.resources.requests
  field := concat(".", [spec.field, "containers", string(index), "resources"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing resource requests",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Limits" doesn’t have field "cpu"
lint[output] {
  rule_name := "resource-limits-cpu"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.limits
  not container.resources.limits.cpu
  field := concat(".", [spec.field, "containers", string(index), "resources", "limits"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing resource cpu limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Limits" doesn’t have field "memory"
lint[output] {
  rule_name := "resource-limits-memory"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.limits
  not container.resources.limits.memory
  field := concat(".", [spec.field, "containers", string(index), "resources", "limits"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing resource memory limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Requests" doesn’t have field "cpu"
lint[output] {
  rule_name := "resource-requests-cpu"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.requests
  not container.resources.requests.cpu
  field := concat(".", [spec.field, "containers", string(index), "resources", "requests"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing requests cpu limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Requests" doesn’t have field "memory"
lint[output] {
  rule_name := "resource-requests-memory"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.requests
  not container.resources.requests.memory
  field := concat(".", [spec.field, "containers", string(index), "resources", "requests"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Missing requests memory limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Volume" of a spec has field "hostPath"
lint[output] {
  rule_name := "volumes-host-paths"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  volume := spec.spec.volumes[index]
  volume.hostPath
  field := concat(".", [spec.field, "volumes", string(index), "hostPath"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Volume has hostpath",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Volume" of a spec has field "hostPath" set to "docker.sock"
lint[output] {
  rule_name := "volume-docker-sock"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  spec := specs[_]
  volume := spec.spec.volumes[index]
  volume.hostPath.path == "/var/run/docker.sock"
  field := concat(".", [spec.field, "volumes", string(index), "hostPath", "path"])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Volume mounts docker.sock",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "namespace" is hardcoded
lint[output] {
  rule_name := "hardcoded-namespace"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  file := files[_]
  namespace := file.content.metadata.namespace
  is_string(namespace)
  not re_match("^(repl{{|{{repl)", namespace)
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Found a hardcoded namespace",
    "path": file.path,
    "field": "metadata.namespace",
    "docIndex": file.docIndex
  }
}

# Check if any file may contain secrets
lint[output] {
  rule_name := "may-contain-secrets"
  rule_config := lint_rule_config(rule_name, "info")
  not rule_config.off
  file := input[_] # using "input" instead if "files" because "file.content" is string in "input"
  expression := secrets_regular_expressions[_]
  expression_matches := regex.find_n(expression, file.content, -1)
  count(expression_matches) > 0
  match := expression_matches[_]
  not re_match("repl{{|{{repl", match) # exclude if template function
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "It looks like there might be secrets in this file",
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": match
  }
}

# Check if ConfigOption has a valid type
lint[output] {
  rule_name := "config-option-invalid-type"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.type
  is_string(item.type)
  not re_match("^(text|label|password|file|bool|select_one|select_many|textarea|select|heading|radio|dropdown)$", item.type)
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Config option \"%s\" has an invalid type", [string(item.name)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if repeatable ConfigOption has a template field defined
lint[output] {
  rule_name := "repeat-option-missing-template"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.repeatable
  not item.templates
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Repeatable Config option \"%s\" has an incomplete template target", [string(item.name)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if repeatable ConfigOption has a valuesByGroup field
lint[output] {
  rule_name := "repeat-option-missing-valuesByGroup"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.repeatable
  not item.valuesByGroup
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Repeatable Config option \"%s\" has an incomplete valuesByGroup", [string(item.name)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if repeatable ConfigOption template ends in array
lint[output] {
  rule_name := "repeat-option-malformed-yamlpath"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.repeatable
  template := item.templates[_]
  template.yamlPath
  not template_yamlPath_ends_with_array(template)
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Repeatable Config option \"%s\" yamlPath does not end with an array", [string(item.name)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if ConfigOption should have a "password" type
lint[output] {
  rule_name := "config-option-password-type"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  is_string(item.name)
  re_match("password|secret|token", item.name)
  item.type != "password"
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Config option \"%s\" should have type \"password\"", [item.name])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if all ConfigOptions exist
lint[output] {
  rule_name := "config-option-not-found"
  rule_config := lint_rule_config(rule_name, "warn")
  not rule_config.off

  file := input[_]

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)?([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, file.content, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  not config_option_exists(option_name)

  message := sprintf("Config option \"%s\" not found", [option_name])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": capture_groups[0]
  }
}

# Check if ConfigOption is circular (references itself)
lint[output] {
  rule_name := "config-option-is-circular"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off

  config_option := config_options[_]
  item := config_option.item
  value := item[key]

  key != "items"

  marshalled_value := yaml.marshal(value)

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)?([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, marshalled_value, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  item.name == option_name

  field := concat(".", [config_option.field, string(key)])

  message := sprintf("Config option \"%s\" references itself", [option_name])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if sub-templated ConfigOptions are repeatable
lint[output] {
  rule_name := "config-option-not-repeatable"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off

  file := input[_]

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, file.content, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  not config_option_is_repeatable(option_name)

  message := sprintf("Config option \"%s\" not repeatable", [option_name])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": capture_groups[0]
  }
}

# Check if "when" is valid
is_when_valid(when) {
  is_boolean(when)
} else {
  is_string(when)
  expression := "^((repl{{|{{repl).*[^}]}}$)|([tT]rue|[fF]alse)$"
  re_match(expression, when)
}
lint[output] {
  rule_name := "config-option-when-is-invalid"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item

  not is_when_valid(item.when)

  field := concat(".", [config_option.field, "when"])

  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Invalid \"when\" expression",
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if ConfigOption Regex Validators are valid
lint[output] {
  rule_name := "config-option-invalid-regex-validator"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.validation.regex
  not regex.is_valid(item.validation.regex.pattern)
  field := concat(".", [config_option.field, "validation", "regex", "pattern"])
  message := sprintf("Config option regex validator pattern \"%s\" is invalid", [string(item.validation.regex.pattern)])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if type is one of [text|textarea|password|file] when validation is present
lint[output] {
  rule_name := "config-option-regex-validator-invalid-type"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  config_option := config_options[_]
  item := config_option.item
  item.validation.regex.pattern
  not re_match("text|textarea|password|file", item.type)
  field := concat(".", [config_option.field, "type",])
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "Config option type should be one of [text|textarea|password|file] with regex validator",
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# A function to check that a backup resource exists
v1_backup_spec_exists {
  file := files[_]
  file.content.kind == "Backup"
  file.content.apiVersion == "velero.io/v1"
}
# A function to check that a restore resource exists
v1_restore_spec_exists {
  file := files[_]
  file.content.kind == "Restore"
  file.content.apiVersion == "velero.io/v1"
}
# A rule that returns the restore file path
restore_file_path = file.path {
  file := files[_]
  file.content.kind == "Restore"
  file.content.apiVersion == "velero.io/v1"
}

# Validate that a velero backup resource exists when a velero restore resource is present
lint[output] {
  rule_name := "backup-resource-required-when-restore-exists"
  rule_config := lint_rule_config(rule_name, "error")
  not rule_config.off
  not v1_backup_spec_exists
  v1_restore_spec_exists
  output := {
    "rule": rule_name,
    "type": rule_config.level,
    "message": "A velero backup resource is required when a velero restore resource is included",
    "path": restore_file_path,
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
