package api

import (
	"os"

	"github.com/replicatedhq/ship/pkg/api/amazoneks"
)

// Assets is the top level assets object
type Assets struct {
	V1 []Asset `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// AssetShared is attributes common to all assets
type AssetShared struct {
	// Dest is where this file should be output
	Dest string `json:"dest,omitempty" yaml:"dest,omitempty" hcl:"dest,omitempty"`
	// Mode is where this file should be output
	Mode os.FileMode `json:"mode,omitempty" yaml:"mode,omitempty" hcl:"mode,omitempty"`
	// Description is an optional description
	Description string `json:"description,omitempty" yaml:"description,omitempty" hcl:"description,omitempty"`
	// When is an optional boolean to determine whether to pull asset
	When string `json:"when,omitempty" yaml:"when,omitempty" hcl:"when,omitempty"`
}

// Asset is a spec to generate one or more deployment assets
type Asset struct {
	Inline      *InlineAsset      `json:"inline,omitempty" yaml:"inline,omitempty" hcl:"inline,omitempty"`
	Docker      *DockerAsset      `json:"docker,omitempty" yaml:"docker,omitempty" hcl:"docker,omitempty"`
	DockerLayer *DockerLayerAsset `json:"dockerlayer,omitempty" yaml:"dockerlayer,omitempty" hcl:"dockerlayer,omitempty"`
	GitHub      *GitHubAsset      `json:"github,omitempty" yaml:"github,omitempty" hcl:"github,omitempty"`
	Web         *WebAsset         `json:"web,omitempty" yaml:"web,omitempty" hcl:"web,omitempty"`
	Helm        *HelmAsset        `json:"helm,omitempty" yaml:"helm,omitempty" hcl:"helm,omitempty"`
	Local       *LocalAsset       `json:"local,omitempty" yaml:"local,omitempty" hcl:"local,omitempty"`
	Terraform   *TerraformAsset   `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	AmazonEKS   *EKSAsset         `json:"amazon_eks,omitempty" yaml:"amazon_eks,omitempty" hcl:"amazon_eks,omitempty"`
	GoogleGKE   *GKEAsset         `json:"google_gke,omitempty" yaml:"google_gke,omitempty" hcl:"google_gke,omitempty"`
	AzureAKS    *AKSAsset         `json:"azure_aks,omitempty" yaml:"azure_aks,omitempty" hcl:"azure_aks,omitempty"`
}

// InlineAsset is an asset whose contents are specified directly in the Spec
type InlineAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Contents    string `json:"contents" yaml:"contents" hcl:"contents"`
}

// DockerAsset is an asset that declares a docker image
type DockerAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Image       string `json:"image" yaml:"image" hcl:"image"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
}

// DockerLayerAsset is an asset that will unpack a docker layer at `dest`
type DockerLayerAsset struct {
	DockerAsset `json:",inline" yaml:",inline" hcl:",inline"`
	Layer       string `json:"layer" yaml:"layer" hcl:"layer"`
}

// GitHubAsset is an asset whose contents are specified directly in the Spec
type GitHubAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Repo        string `json:"repo" yaml:"repo" hcl:"repo"`
	Ref         string `json:"ref" yaml:"ref" hcl:"ref"`
	Path        string `json:"path" yaml:"path" hcl:"path"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
	Proxy       bool   `json:"proxy" yaml:"proxy" hcl:"proxy"`
	StripPath   string `json:"strip_path" yaml:"strip_path" hcl:"strip_path"`
}

// WebAsset is an asset whose contents are specified by the HTML at the corresponding URL
type WebAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Body        string              `json:"body" yaml:"body" hcl:"body"`
	BodyFormat  string              `json:"bodyFormat" yaml:"bodyFormat" hcl:"bodyFormat"`
	Headers     map[string][]string `json:"headers" yaml:"headers" hcl:"headers"`
	Method      string              `json:"method" yaml:"method" hcl:"method"`
	URL         string              `json:"url" yaml:"url" hcl:"url"`
}

// HelmAsset is an asset that declares a helm chart on github
type HelmAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Values      map[string]interface{} `json:"values" yaml:"values" hcl:"values"`
	HelmOpts    []string               `json:"helm_opts" yaml:"helm_opts" hcl:"helm_opts"`
	// GitHub references a github asset from which to pull the chart
	GitHub *GitHubAsset `json:"github,omitempty" yaml:"github,omitempty" hcl:"github,omitempty"`
	// HelmFetch pulls a chart as 'helm fetch' would
	HelmFetch *HelmFetch `json:"helm_fetch,omitempty" yaml:"helm_fetch,omitempty" hcl:"helm_fetch,omitempty"`
	// Local is an escape hatch, most impls will use github or some sort of ChartMuseum thing
	Local      *LocalHelmOpts `json:"local,omitempty" yaml:"local,omitempty" hcl:"local,omitempty"`
	ValuesFrom *ValuesFrom    `json:"values_from,omitempty" yaml:"values_from,omitempty" hcl:"values_from,omitempty"`
	Upstream   string         `json:"upstream,omitempty" yaml:"upstream,omitempty" hcl:"upstream,omitempty"`
}

// LocalAsset is an asset whose contents are on the local fs
type LocalAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Path        string `json:"path" yaml:"path" hcl:"path"`
}

type ValuesFrom struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty" hcl:"path,omitempty"`
	// SaveToState is used when a HelmValues step is not part of the lifecycle (e.g. Unfork) in order to save
	// the merged helm values to state.
	SaveToState bool `json:"save_to_state,omitempty" yaml:"save_to_state,omitempty" hcl:"save_to_state,omitempty"`
}

type ValuesFromLifecycle struct{}

// LocalHelmOpts specifies a helm chart that should be templated
// using other assets that are already present at `ChartRoot`
type LocalHelmOpts struct {
	ChartRoot string `json:"chart_root" yaml:"chart_root" hcl:"chart_root"`
}

type HelmFetch struct {
	ChartRef string `json:"chart_ref" yaml:"chart_ref" hcl:"chart_ref"`
	RepoURL  string `json:"repo_url" yaml:"repo_url" hcl:"repo_url"`
	Version  string `json:"version" yaml:"version" hcl:"version"`
}

// TerraformAsset
type TerraformAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	// GitHub references a github asset from which to pull a terraform module
	GitHub *GitHubAsset `json:"github" yaml:"github" hcl:"github"`
	// Inline allows a vendor to specify a terraform module inline in ship
	Inline string `json:"inline,omitempty" yaml:"inline,omitempty" hcl:"inline,omitempty"`
}

// EKSAsset
type EKSAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`

	ClusterName string `json:"cluster_name,omitempty" yaml:"cluster_name,omitempty" hcl:"cluster_name,omitempty"`
	Region      string `json:"region,omitempty" yaml:"region,omitempty" hcl:"region,omitempty"`

	CreatedVPC        *amazoneks.EKSCreatedVPC        `json:"created_vpc,omitempty" yaml:"created_vpc,omitempty" hcl:"created_vpc,omitempty"`
	ExistingVPC       *amazoneks.EKSExistingVPC       `json:"existing_vpc,omitempty" yaml:"existing_vpc,omitempty" hcl:"existing_vpc,omitempty"`
	AutoscalingGroups []amazoneks.EKSAutoscalingGroup `json:"autoscaling_groups,omitempty" yaml:"autoscaling_groups,omitempty" hcl:"autoscaling_groups,omitempty"`
}

// GKEAsset
type GKEAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`

	GCPProvider `json:",inline" yaml:",inline" hcl:",inline"`

	// ClusterName required
	ClusterName      string `json:"cluster_name" yaml:"cluster_name" hcl:"cluster_name"`
	Zone             string `json:"zone,omitempty" yaml:"zone,omitempty" hcl:"zone,omitempty"`
	InitialNodeCount string `json:"initial_node_count,omitempty" yaml:"initial_node_count,omitempty" hcl:"initial_node_count,omitempty"`
	MachineType      string `json:"machine_type,omitempty" yaml:"machine_type,omitempty" hcl:"machine_type,omitempty"`
	AdditionalZones  string `json:"additional_zones,omitempty" yaml:"additional_zones,omitempty" hcl:"additional_zones,omitempty"`
	MinMasterVersion string `json:"min_master_version,omitempty" yaml:"min_master_version,omitempty" hcl:"min_master_version,omitempty"`
}

type GCPProvider struct {
	Credentials string `json:"credentials,omitempty" yaml:"credentials,omitempty" hcl:"credentials,omitempty"`
	Project     string `json:"project,omitempty" yaml:"project,omitempty" hcl:"project,omitempty"`
	Region      string `json:"region,omitempty" yaml:"region,omitempty" hcl:"region,omitempty"`
}

// AKSAsset
type AKSAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Azure       `json:",inline" yaml:",inline" hcl:",inline"`

	ClusterName       string `json:"cluster_name" yaml:"cluster_name" hcl:"cluster_name"`
	KubernetesVersion string `json:"kubernetes_version" yaml:"kubernetes_version" hcl:"kubernetes_version"`
	PublicKey         string `json:"public_key" yaml:"public_key" hcl:"public_key"`
	NodeCount         string `json:"node_count" yaml:"node_count" hcl:"node_count"`
	NodeType          string `json:"node_type" yaml:"node_type" hcl:"node_type"`
	DiskGB            string `json:"disk_gb" yaml:"disk_gb" hcl:"disk_gb"`
}

type Azure struct {
	TenantID               string `json:"tenant_id" yaml:"tenant_id" hcl:"tenant_id"`
	SubscriptionID         string `json:"subscription_id" yaml:"subscription_id" hcl:"subscription_id"`
	ServicePrincipalID     string `json:"service_principal_id" yaml:"service_principal_id" hcl:"service_principal_id"`
	ServicePrincipalSecret string `json:"service_principal_secret" yaml:"service_principal_secret" hcl:"service_principal_secret"`
	ResourceGroupName      string `json:"resource_group_name" yaml:"resource_group_name" hcl:"resource_group_name"`
	Location               string `json:"location" yaml:"location" hcl:"location"`
}
