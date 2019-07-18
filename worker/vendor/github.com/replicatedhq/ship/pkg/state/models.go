package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/terraform"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/replicatedhq/ship/pkg/version"
)

type State struct {
	V1 *V1 `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

func (v State) IsEmpty() bool {
	return v.V1 == nil
}

type V1 struct {
	Config             map[string]interface{} `json:"config" yaml:"config" hcl:"config"`
	Terraform          *Terraform             `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	HelmValues         string                 `json:"helmValues,omitempty" yaml:"helmValues,omitempty" hcl:"helmValues,omitempty"`
	ReleaseName        string                 `json:"releaseName,omitempty" yaml:"releaseName,omitempty" hcl:"releaseName,omitempty"`
	Namespace          string                 `json:"namespace,omitempty" yaml:"namespace,omitempty" hcl:"namespace,omitempty"`
	HelmValuesDefaults string                 `json:"helmValuesDefaults,omitempty" yaml:"helmValuesDefaults,omitempty" hcl:"helmValuesDefaults,omitempty"`
	Kustomize          *Kustomize             `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
	Upstream           string                 `json:"upstream,omitempty" yaml:"upstream,omitempty" hcl:"upstream,omitempty"`
	Metadata           *Metadata              `json:"metadata,omitempty" yaml:"metadata,omitempty" hcl:"metadata,omitempty"`
	UpstreamContents   *UpstreamContents      `json:"upstreamContents,omitempty" yaml:"upstreamContents,omitempty" hcl:"upstreamContents,omitempty"`
	ShipVersion        *version.Build         `json:"shipVersion,omitempty" yaml:"shipVersion,omitempty" hcl:"shipVersion,omitempty"`

	//deprecated in favor of upstream
	ChartURL string `json:"chartURL,omitempty" yaml:"chartURL,omitempty" hcl:"chartURL,omitempty"`

	ChartRepoURL string    `json:"ChartRepoURL,omitempty" yaml:"ChartRepoURL,omitempty" hcl:"ChartRepoURL,omitempty"`
	ChartVersion string    `json:"ChartVersion,omitempty" yaml:"ChartVersion,omitempty" hcl:"ChartVersion,omitempty"`
	ContentSHA   string    `json:"contentSHA,omitempty" yaml:"contentSHA,omitempty" hcl:"contentSHA,omitempty"`
	Lifecycle    *Lifeycle `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty" hcl:"lifecycle,omitempty"`

	CAs   map[string]util.CAType   `json:"cas,omitempty" yaml:"cas,omitempty" hcl:"cas,omitempty"`
	Certs map[string]util.CertType `json:"certs,omitempty" yaml:"certs,omitempty" hcl:"certs,omitempty"`
}

type License struct {
	ID        string    `json:"id" yaml:"id" hcl:"id"`
	Assignee  string    `json:"assignee" yaml:"assignee" hcl:"assignee"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt" hcl:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt" yaml:"expiresAt" hcl:"expiresAt"`
	Type      string    `json:"type" yaml:"type" hcl:"type"`
}

type Metadata struct {
	ApplicationType string  `json:"applicationType" yaml:"applicationType" hcl:"applicationType"`
	Sequence        int64   `json:"sequence" yaml:"sequence" hcl:"sequence" meta:"sequence"`
	Icon            string  `json:"icon,omitempty" yaml:"icon,omitempty" hcl:"icon,omitempty"`
	Name            string  `json:"name,omitempty" yaml:"name,omitempty" hcl:"name,omitempty"`
	ReleaseNotes    string  `json:"releaseNotes" yaml:"releaseNotes" hcl:"releaseNotes"`
	Version         string  `json:"version" yaml:"version" hcl:"version"`
	CustomerID      string  `json:"customerID,omitempty" yaml:"customerID,omitempty" hcl:"customerID,omitempty"`
	InstallationID  string  `json:"installationID,omitempty" yaml:"installationID,omitempty" hcl:"installationID,omitempty"`
	LicenseID       string  `json:"licenseID,omitempty" yaml:"licenseID,omitempty" hcl:"licenseID,omitempty"`
	AppSlug         string  `json:"appSlug,omitempty" yaml:"appSlug,omitempty" hcl:"appSlug,omitempty"`
	License         License `json:"license" yaml:"license" hcl:"license"`
}

type UpstreamContents struct {
	UpstreamFiles []UpstreamFile `json:"upstreamFiles,omitempty" yaml:"upstreamFiles,omitempty" hcl:"upstreamFiles,omitempty"`
	AppRelease    *ShipRelease   `json:"appRelease,omitempty" yaml:"appRelease,omitempty" hcl:"appRelease,omitempty"`
}

type UpstreamFile struct {
	FilePath     string `json:"filePath,omitempty" yaml:"filePath,omitempty" hcl:"filePath,omitempty"`
	FileContents string `json:"fileContents,omitempty" yaml:"fileContents,omitempty" hcl:"fileContents,omitempty"`
}

type StepsCompleted map[string]interface{}

func (s StepsCompleted) String() string {
	acc := new(bytes.Buffer)
	for key := range s {
		fmt.Fprintf(acc, "%s;", key)
	}
	return acc.String()

}

type Lifeycle struct {
	StepsCompleted StepsCompleted `json:"stepsCompleted,omitempty" yaml:"stepsCompleted,omitempty" hcl:"stepsCompleted,omitempty"`
}

func (l *Lifeycle) WithCompletedStep(step api.Step) *Lifeycle {
	updated := &Lifeycle{StepsCompleted: map[string]interface{}{}}
	if l != nil && l.StepsCompleted != nil {
		updated.StepsCompleted = l.StepsCompleted
	}

	updated.StepsCompleted[step.Shared().ID] = true
	for _, nowInvalid := range step.Shared().Invalidates {
		delete(updated.StepsCompleted, nowInvalid)
	}
	return updated
}

type Overlay struct {
	ExcludedBases     []string          `json:"excludedBases,omitempty" yaml:"excludedBases,omitempty" hcl:"excludedBases,omitempty"`
	Patches           map[string]string `json:"patches,omitempty" yaml:"patches,omitempty" hcl:"patches,omitempty"`
	Resources         map[string]string `json:"resources,omitempty" yaml:"resources,omitempty" hcl:"resources,omitempty"`
	KustomizationYAML string            `json:"kustomization_yaml,omitempty" yaml:"kustomization_yaml,omitempty" hcl:"kustomization_yaml,omitempty"`
}

func NewOverlay() Overlay {
	return Overlay{
		ExcludedBases: []string{},
		Patches:       map[string]string{},
		Resources:     map[string]string{},
	}
}

type Kustomize struct {
	Overlays map[string]Overlay `json:"overlays,omitempty" yaml:"overlays,omitempty" hcl:"overlays,omitempty"`
}

func (k *Kustomize) Ship() Overlay {
	if k.Overlays == nil {
		return NewOverlay()
	}
	if ship, ok := k.Overlays["ship"]; ok {
		return ship
	}

	return NewOverlay()
}

func (v State) CurrentKustomize() *Kustomize {
	if v.V1 != nil {
		return v.V1.Kustomize
	}
	return nil
}

func (v State) CurrentKustomizeOverlay(filename string) (contents string, isResource bool) {
	if v.V1.Kustomize == nil {
		return
	}

	if v.V1.Kustomize.Overlays == nil {
		return
	}

	overlay, ok := v.V1.Kustomize.Overlays["ship"]
	if !ok {
		return
	}

	if overlay.Patches != nil {
		file, ok := overlay.Patches[filename]
		if ok {
			return file, false
		}
	}

	if overlay.Resources != nil {
		file, ok := overlay.Resources[filename]
		if ok {
			return file, true
		}
	}
	return
}

type Terraform struct {
	RawState string           `json:"rawState,omitempty" yaml:"rawState,omitempty" hcl:"rawState,omitempty"`
	State    *terraform.State `json:"state,omitempty" yaml:"state,omitempty" hcl:"state,omitempty"`
}

func (v State) CurrentConfig() (map[string]interface{}, error) {
	if v.V1 == nil || v.V1.Config == nil {
		return make(map[string]interface{}), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(v.V1.Config)
	if err != nil {
		return nil, err
	}
	var configClone map[string]interface{}
	err = dec.Decode(&configClone)
	if err != nil {
		return nil, err
	}
	return configClone, nil
}

func (v State) CurrentHelmValues() string {
	if v.V1 != nil {
		return v.V1.HelmValues
	}
	return ""
}

func (v State) CurrentHelmValuesDefaults() string {
	if v.V1 != nil {
		return v.V1.HelmValuesDefaults
	}
	return ""
}

func (v State) CurrentReleaseName() string {
	if v.V1 != nil {
		return v.V1.ReleaseName
	}
	return ""
}

func (v State) CurrentNamespace() string {
	if v.V1 != nil {
		return v.V1.Namespace
	}
	return ""
}

func (v State) Upstream() string {
	if v.V1 != nil {
		if v.V1.Upstream != "" {
			return v.V1.Upstream
		}
		return v.V1.ChartURL
	}
	return ""
}

func (v State) Versioned() State {
	if v.V1 == nil {
		v.V1 = &V1{}
	}
	return v
}

func (v State) WithCompletedStep(step api.Step) State {
	v.V1.Lifecycle = v.V1.Lifecycle.WithCompletedStep(step)
	return v
}

func (v State) migrateDeprecatedFields() State {
	if v.V1 != nil {
		v.V1.Upstream = v.Upstream()
		v.V1.ChartURL = ""
	}
	return v
}

func (v State) CurrentCAs() map[string]util.CAType {
	if v.V1 != nil {
		return v.V1.CAs
	}
	return nil
}

func (v State) CurrentCerts() map[string]util.CertType {
	if v.V1 != nil {
		return v.V1.Certs
	}
	return nil
}

func (v State) UpstreamContents() *UpstreamContents {
	if v.V1 != nil {
		if v.V1.UpstreamContents != nil {
			return v.V1.UpstreamContents
		}
		return nil
	}
	return nil
}

type Image struct {
	URL      string `json:"url"`
	Source   string `json:"source"`
	AppSlug  string `json:"appSlug"`
	ImageKey string `json:"imageKey"`
}

type GithubFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Sha  string `json:"sha"`
	Size int64  `json:"size"`
	Data string `json:"data"`
}

type GithubContent struct {
	Repo  string       `json:"repo"`
	Path  string       `json:"path"`
	Ref   string       `json:"ref"`
	Files []GithubFile `json:"files"`
}

// ShipRelease is the release response from GQL
type ShipRelease struct {
	ID             string           `json:"id"`
	Sequence       int64            `json:"sequence"`
	ChannelID      string           `json:"channelId"`
	ChannelName    string           `json:"channelName"`
	ChannelIcon    string           `json:"channelIcon"`
	Semver         string           `json:"semver"`
	ReleaseNotes   string           `json:"releaseNotes"`
	Spec           string           `json:"spec"`
	Images         []Image          `json:"images"`
	GithubContents []GithubContent  `json:"githubContents"`
	Created        string           `json:"created"` // TODO: this time is not in RFC 3339 format
	RegistrySecret string           `json:"registrySecret,omitempty"`
	Entitlements   api.Entitlements `json:"entitlements,omitempty"`
	CollectSpec    string           `json:"collectSpec,omitempty"`
	AnalyzeSpec    string           `json:"analyzeSpec,omitempty"`
}

// ToReleaseMeta linter
func (r *ShipRelease) ToReleaseMeta() api.ReleaseMetadata {
	return api.ReleaseMetadata{
		ReleaseID:      r.ID,
		Sequence:       r.Sequence,
		ChannelID:      r.ChannelID,
		ChannelName:    r.ChannelName,
		ChannelIcon:    r.ChannelIcon,
		Semver:         r.Semver,
		ReleaseNotes:   r.ReleaseNotes,
		Created:        r.Created,
		RegistrySecret: r.RegistrySecret,
		Images:         r.apiImages(),
		GithubContents: r.githubContents(),
		Entitlements:   r.Entitlements,
		CollectSpec:    r.CollectSpec,
		AnalyzeSpec:    r.AnalyzeSpec,
	}
}

func (r *ShipRelease) apiImages() []api.Image {
	result := []api.Image{}
	for _, image := range r.Images {
		result = append(result, api.Image(image))
	}
	return result
}

func (r *ShipRelease) githubContents() []api.GithubContent {
	result := []api.GithubContent{}
	for _, content := range r.GithubContents {
		files := []api.GithubFile{}
		for _, file := range content.Files {
			files = append(files, api.GithubFile(file))
		}
		apiCont := api.GithubContent{
			Repo:  content.Repo,
			Path:  content.Path,
			Ref:   content.Ref,
			Files: files,
		}
		result = append(result, apiCont)
	}
	return result
}

func (v State) ReleaseMetadata() *api.ReleaseMetadata {
	if v.V1 != nil {
		if v.V1.UpstreamContents != nil {
			baseMeta := v.V1.UpstreamContents.AppRelease.ToReleaseMeta()
			if v.V1.Metadata != nil {
				baseMeta.CustomerID = v.V1.Metadata.CustomerID
				baseMeta.InstallationID = v.V1.Metadata.InstallationID
				baseMeta.LicenseID = v.V1.Metadata.LicenseID
				baseMeta.AppSlug = v.V1.Metadata.AppSlug
				baseMeta.License.ID = v.V1.Metadata.License.ID
				baseMeta.License.Assignee = v.V1.Metadata.License.Assignee
				baseMeta.License.CreatedAt = v.V1.Metadata.License.CreatedAt
				baseMeta.License.ExpiresAt = v.V1.Metadata.License.ExpiresAt
				baseMeta.License.Type = v.V1.Metadata.License.Type
			}
			return &baseMeta
		}
		return nil
	}
	return nil
}
