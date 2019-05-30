package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/replicatedhq/ship/pkg/constants"
)

var releaseNameRegex = regexp.MustCompile("[^a-zA-Z0-9\\-]")

// Spec is the top level Ship document that defines an application
type Spec struct {
	Assets    Assets    `json:"assets" yaml:"assets" hcl:"asset"`
	Lifecycle Lifecycle `json:"lifecycle" yaml:"lifecycle" hcl:"lifecycle"`
	Config    Config    `json:"config" yaml:"config" hcl:"config"`
}

// Image
type Image struct {
	URL      string `json:"url" yaml:"url" hcl:"url" meta:"url"`
	Source   string `json:"source" yaml:"source" hcl:"source" meta:"source"`
	AppSlug  string `json:"appSlug" yaml:"appSlug" hcl:"appSlug" meta:"appSlug"`
	ImageKey string `json:"imageKey" yaml:"imageKey" hcl:"imageKey" meta:"imageKey"`
}

type GithubContent struct {
	Repo  string       `json:"repo" yaml:"repo" hcl:"repo" meta:"repo"`
	Path  string       `json:"path" yaml:"path" hcl:"path" meta:"path"`
	Ref   string       `json:"ref" yaml:"ref" hcl:"ref" meta:"ref"`
	Files []GithubFile `json:"files" yaml:"files" hcl:"files" meta:"files"`
}

// Not using json.Marshal because I want to omit the file data, and don't feel like
// writing a custom marshaller
func (g GithubContent) String() string {

	fileStr := "["
	for _, file := range g.Files {
		fileStr += fmt.Sprintf("%s, ", file.String())
	}
	fileStr += "]"

	return fmt.Sprintf("GithubContent{ repo:%s path:%s ref:%s files:%s }", g.Repo, g.Path, g.Ref, fileStr)
}

// GithubFile
type GithubFile struct {
	Name string `json:"name" yaml:"name" hcl:"name" meta:"name"`
	Path string `json:"path" yaml:"path" hcl:"path" meta:"path"`
	Sha  string `json:"sha" yaml:"sha" hcl:"sha" meta:"sha"`
	Size int64  `json:"size" yaml:"size" hcl:"size" meta:"size"`
	Data string `json:"data" yaml:"data" hcl:"data" meta:"data"`
}

func (file GithubFile) String() string {
	return fmt.Sprintf("GitHubFile{ name:%s path:%s sha:%s size:%d dataLen:%d }",
		file.Name, file.Path, file.Sha, file.Size, len(file.Data))

}

type ShipAppMetadata struct {
	Description  string `json:"description" yaml:"description" hcl:"description" meta:"description"`
	Version      string `json:"version" yaml:"version" hcl:"version" meta:"version"`
	Icon         string `json:"icon" yaml:"icon" hcl:"icon" meta:"icon"`
	Name         string `json:"name" yaml:"name" hcl:"name" meta:"name"`
	Readme       string `json:"readme" yaml:"readme" hcl:"readme" meta:"readme"`
	URL          string `json:"url" yaml:"url" hcl:"url" meta:"url"`
	ContentSHA   string `json:"contentSHA" yaml:"contentSHA" hcl:"contentSHA" meta:"contentSHA"`
	ReleaseNotes string `json:"releaseNotes" yaml:"releaseNotes" hcl:"releaseNotes" meta:"release-notes"`
}

type License struct {
	ID        string    `json:"id" yaml:"id" hcl:"id" meta:"id"`
	Assignee  string    `json:"assignee" yaml:"assignee" hcl:"assignee" meta:"assignee"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt" hcl:"createdAt" meta:"created-at"`
	ExpiresAt time.Time `json:"expiresAt" yaml:"expiresAt" hcl:"expiresAt" meta:"expires-at"`
	Type      string    `json:"type" yaml:"type" hcl:"type" meta:"type"`
}

// ReleaseMetadata
type ReleaseMetadata struct {
	ReleaseID       string          `json:"releaseId" yaml:"releaseId" hcl:"releaseId" meta:"release-id"`
	Sequence        int64           `json:"sequence" yaml:"sequence" hcl:"sequence" meta:"sequence"`
	CustomerID      string          `json:"customerId" yaml:"customerId" hcl:"customerId" meta:"customer-id"`
	InstallationID  string          `json:"installation" yaml:"installation" hcl:"installation" meta:"installation-id"`
	ChannelID       string          `json:"channelId" yaml:"channelId" hcl:"channelId" meta:"channel-id"`
	AppSlug         string          `json:"appSlug" yaml:"appSlug" hcl:"appSlug" meta:"app-slug"`
	LicenseID       string          `json:"licenseId" yaml:"licenseId" hcl:"licenseId" meta:"license-id"`
	ChannelName     string          `json:"channelName" yaml:"channelName" hcl:"channelName" meta:"channel-name"`
	ChannelIcon     string          `json:"channelIcon" yaml:"channelIcon" hcl:"channelIcon" meta:"channel-icon"`
	Semver          string          `json:"semver" yaml:"semver" hcl:"semver" meta:"release-version"`
	ReleaseNotes    string          `json:"releaseNotes" yaml:"releaseNotes" hcl:"releaseNotes" meta:"release-notes"`
	Created         string          `json:"created" yaml:"created" hcl:"created" meta:"release-date"`
	Installed       string          `json:"installed" yaml:"installed" hcl:"installed" meta:"install-date"`
	RegistrySecret  string          `json:"registrySecret" yaml:"registrySecret" hcl:"registrySecret" meta:"registry-secret"`
	Images          []Image         `json:"images" yaml:"images" hcl:"images" meta:"images"`
	GithubContents  []GithubContent `json:"githubContents" yaml:"githubContents" hcl:"githubContents" meta:"githubContents"`
	ShipAppMetadata ShipAppMetadata `json:"shipAppMetadata" yaml:"shipAppMetadata" hcl:"shipAppMetadata" meta:"shipAppMetadata"`
	Entitlements    Entitlements    `json:"entitlements" yaml:"entitlements" hcl:"entitlements" meta:"entitlements"`
	Type            string          `json:"type" yaml:"type" hcl:"type" meta:"type"`
	License         License         `json:"license" yaml:"license" hcl:"license" meta:"license"`
}

func (r ReleaseMetadata) ReleaseName() string {
	var releaseName string

	if r.ChannelName != "" {
		releaseName = r.ChannelName
	}

	if r.ShipAppMetadata.Name != "" {
		releaseName = r.ShipAppMetadata.Name
	}

	if releaseName == "" && r.AppSlug != "" {
		releaseName = r.AppSlug
	}

	if len(releaseName) == 0 {
		return "ship"
	}

	releaseName = strings.ToLower(releaseName)
	return releaseNameRegex.ReplaceAllLiteralString(releaseName, "-")
}

// Release
type Release struct {
	Metadata ReleaseMetadata
	Spec     Spec
}

func (r *Release) FindRenderStep() *Render {
	for _, step := range r.Spec.Lifecycle.V1 {
		if step.Render != nil {
			return step.Render
		}
	}
	return nil
}

func (r *Release) FindRenderRoot() string {
	render := r.FindRenderStep()
	if render == nil {
		return constants.InstallerPrefixPath
	}

	return render.RenderRoot()
}
