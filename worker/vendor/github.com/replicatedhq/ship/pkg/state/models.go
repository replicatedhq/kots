package state

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/terraform"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
)

// now that we have Versioned(), we probably don't need nearly so broad an interface here
type State interface {
	CurrentConfig() map[string]interface{}
	CurrentKustomize() *Kustomize
	CurrentKustomizeOverlay(filename string) (string, bool)
	CurrentHelmValues() string
	CurrentHelmValuesDefaults() string
	CurrentReleaseName() string
	CurrentNamespace() string
	Upstream() string
	Versioned() VersionedState
	IsEmpty() bool
	CurrentCAs() map[string]util.CAType
	CurrentCerts() map[string]util.CertType
}

var _ State = VersionedState{}
var _ State = Empty{}
var _ State = V0{}

type Empty struct{}

func (Empty) CurrentKustomize() *Kustomize                  { return nil }
func (Empty) CurrentKustomizeOverlay(string) (string, bool) { return "", false }
func (Empty) CurrentConfig() map[string]interface{}         { return make(map[string]interface{}) }
func (Empty) CurrentHelmValues() string                     { return "" }
func (Empty) CurrentHelmValuesDefaults() string             { return "" }
func (Empty) CurrentReleaseName() string                    { return "" }
func (Empty) CurrentNamespace() string                      { return "" }
func (Empty) CurrentCAs() map[string]util.CAType            { return nil }
func (Empty) CurrentCerts() map[string]util.CertType        { return nil }
func (Empty) Upstream() string                              { return "" }
func (Empty) Versioned() VersionedState                     { return VersionedState{V1: &V1{}} }
func (Empty) IsEmpty() bool                                 { return true }

type V0 map[string]interface{}

func (v V0) CurrentConfig() map[string]interface{}         { return v }
func (v V0) CurrentKustomize() *Kustomize                  { return nil }
func (v V0) CurrentKustomizeOverlay(string) (string, bool) { return "", false }
func (v V0) CurrentHelmValues() string                     { return "" }
func (v V0) CurrentHelmValuesDefaults() string             { return "" }
func (v V0) CurrentReleaseName() string                    { return "" }
func (v V0) CurrentNamespace() string                      { return "" }
func (v V0) CurrentCAs() map[string]util.CAType            { return nil }
func (v V0) CurrentCerts() map[string]util.CertType        { return nil }
func (v V0) Upstream() string                              { return "" }
func (v V0) Versioned() VersionedState                     { return VersionedState{V1: &V1{Config: v}} }
func (v V0) IsEmpty() bool                                 { return false }

type VersionedState struct {
	V1 *V1 `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

func (v VersionedState) IsEmpty() bool {
	return false
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

	//deprecated in favor of upstream
	ChartURL string `json:"chartURL,omitempty" yaml:"chartURL,omitempty" hcl:"chartURL,omitempty"`

	ChartRepoURL string    `json:"ChartRepoURL,omitempty" yaml:"ChartRepoURL,omitempty" hcl:"ChartRepoURL,omitempty"`
	ChartVersion string    `json:"ChartVersion,omitempty" yaml:"ChartVersion,omitempty" hcl:"ChartVersion,omitempty"`
	ContentSHA   string    `json:"contentSHA,omitempty" yaml:"contentSHA,omitempty" hcl:"contentSHA,omitempty"`
	Lifecycle    *Lifeycle `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty" hcl:"lifecycle,omitempty"`

	CAs   map[string]util.CAType   `json:"cas,omitempty" yaml:"cas,omitempty" hcl:"cas,omitempty"`
	Certs map[string]util.CertType `json:"certs,omitempty" yaml:"certs,omitempty" hcl:"certs,omitempty"`
}

type Metadata struct {
	ApplicationType string      `json:"applicationType" yaml:"applicationType" hcl:"applicationType"`
	Icon            string      `json:"icon,omitempty" yaml:"icon,omitempty" hcl:"icon,omitempty"`
	Name            string      `json:"name,omitempty" yaml:"name,omitempty" hcl:"name,omitempty"`
	ReleaseNotes    string      `json:"releaseNotes" yaml:"releaseNotes" hcl:"releaseNotes"`
	Version         string      `json:"version" yaml:"version" hcl:"version"`
	CustomerID      string      `json:"customerID,omitempty" yaml:"customerID,omitempty" hcl:"customerID,omitempty"`
	InstallationID  string      `json:"installationID,omitempty" yaml:"installationID,omitempty" hcl:"installationID,omitempty"`
	LicenseID       string      `json:"licenseID,omitempty" yaml:"licenseID,omitempty" hcl:"licenseID,omitempty"`
	AppSlug         string      `json:"appSlug,omitempty" yaml:"appSlug,omitempty" hcl:"appSlug,omitempty"`
	Lists           []util.List `json:"lists,omitempty" yaml:"lists,omitempty" hcl:"lists,omitempty"`
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

func (v VersionedState) CurrentKustomize() *Kustomize {
	if v.V1 != nil {
		return v.V1.Kustomize
	}
	return nil
}

func (v VersionedState) CurrentKustomizeOverlay(filename string) (contents string, isResource bool) {
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

func (v VersionedState) CurrentConfig() map[string]interface{} {
	if v.V1 != nil && v.V1.Config != nil {
		return v.V1.Config
	}
	return make(map[string]interface{})
}

func (v VersionedState) CurrentHelmValues() string {
	if v.V1 != nil {
		return v.V1.HelmValues
	}
	return ""
}

func (v VersionedState) CurrentHelmValuesDefaults() string {
	if v.V1 != nil {
		return v.V1.HelmValuesDefaults
	}
	return ""
}

func (v VersionedState) CurrentReleaseName() string {
	if v.V1 != nil {
		return v.V1.ReleaseName
	}
	return ""
}

func (v VersionedState) CurrentNamespace() string {
	if v.V1 != nil {
		return v.V1.Namespace
	}
	return ""
}

func (v VersionedState) Upstream() string {
	if v.V1 != nil {
		if v.V1.Upstream != "" {
			return v.V1.Upstream
		}
		return v.V1.ChartURL
	}
	return ""
}

func (v VersionedState) Versioned() VersionedState {
	return v
}

func (v VersionedState) WithCompletedStep(step api.Step) VersionedState {
	v.V1.Lifecycle = v.V1.Lifecycle.WithCompletedStep(step)
	return v
}

func (v VersionedState) migrateDeprecatedFields() VersionedState {
	if v.V1 != nil {
		v.V1.Upstream = v.Upstream()
		v.V1.ChartURL = ""
	}
	return v
}

func (v VersionedState) CurrentCAs() map[string]util.CAType {
	if v.V1 != nil {
		return v.V1.CAs
	}
	return nil
}

func (v VersionedState) CurrentCerts() map[string]util.CertType {
	if v.V1 != nil {
		return v.V1.Certs
	}
	return nil
}
