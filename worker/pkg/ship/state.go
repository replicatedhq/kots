package ship

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	shipstate "github.com/replicatedhq/ship/pkg/state"
)

type ReplicatedAppWatch struct {
}

func stateFromData(stateJSON []byte) (*shipstate.State, error) {
	shipState := shipstate.State{}
	if err := json.Unmarshal(stateJSON, &shipState); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	return &shipState, nil
}

func WatchNameFromState(stateJSON []byte) string {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		fmt.Println("failed to parse")
		return "Unknown watch"
	}

	if shipState.V1 == nil {
		fmt.Println("no v1")
		return "Unknown watch"
	}

	if shipState.V1.Metadata != nil {
		if shipState.V1.Metadata.ApplicationType == "replicated.app" {
			return shipState.UpstreamContents().AppRelease.ChannelName
		}

		if shipState.V1.Metadata.Name != "" {
			return shipState.V1.Metadata.Name
		} else {
			return shipState.V1.Metadata.AppSlug
		}
	}

	repoRegex := regexp.MustCompile(`github(?:usercontent)?\.com\/([\w-]+)\/([\w-]+)(?:(?:/tree|/blob)?\/([\w-\._]+))?`)
	// attempt to extract a more human-friendly name than the uri
	if repoRegex.MatchString(shipState.V1.Upstream) {
		matches := repoRegex.FindStringSubmatch(shipState.V1.Upstream)

		if len(matches) >= 3 {
			var repoName, version string
			owner := matches[1]
			repo := matches[2]

			if strings.HasPrefix(repo, owner) {
				repoName = repo
			} else {
				repoName = owner + "/" + repo
			}

			if len(matches) >= 4 {
				version = matches[3]
			}

			if version != "" {
				return repoName + "@" + version
			}

			return repoName
		}
	}

	urlRegex := regexp.MustCompile(`(?:https?://)([\w\.\/\-_]+)`)
	if urlRegex.MatchString(shipState.V1.Upstream) {
		matches := urlRegex.FindStringSubmatch(shipState.V1.Upstream)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return shipState.V1.Upstream
}

func WatchIconFromState(stateJSON []byte) string {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return ""
	}

	icon := ""
	if shipState.V1 != nil && shipState.V1.Metadata != nil {
		icon = shipState.V1.Metadata.Icon
		if shipState.V1.Metadata.ApplicationType == "replicated.app" {
			if shipState.V1.UpstreamContents != nil {
				if shipState.V1.UpstreamContents.AppRelease != nil {
					icon = shipState.V1.UpstreamContents.AppRelease.ChannelIcon
				}
			}
		}
	}

	return icon
}

func WatchVersionFromState(stateJSON []byte) string {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return ""
	}

	versionLabel := "Unknown"
	if shipState.V1 != nil && shipState.V1.Metadata != nil && shipState.V1.Metadata.Version != "" {
		versionLabel = shipState.V1.Metadata.Version
	}

	return versionLabel
}

func ShipClusterMetadataFromState(stateJSON []byte) []byte {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return nil
	}

	marshaledMetadata, err := json.Marshal(shipState.V1.Metadata)
	if err != nil {
		return nil
	}

	return marshaledMetadata
}

func TroubleshootCollectorsFromState(stateJSON []byte) []byte {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return nil
	}

	if shipState.V1 == nil || shipState.V1.UpstreamContents == nil || shipState.V1.UpstreamContents.AppRelease == nil {
		return nil
	}

	return []byte(shipState.V1.UpstreamContents.AppRelease.CollectSpec)
}

func TroubleshootAnalyzersFromState(stateJSON []byte) []byte {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return nil
	}

	if shipState.V1 == nil || shipState.V1.UpstreamContents == nil || shipState.V1.UpstreamContents.AppRelease == nil {
		return nil
	}

	return []byte(shipState.V1.UpstreamContents.AppRelease.AnalyzeSpec)
}

func LicenseFromState(stateJSON []byte) []byte {
	shipState, err := stateFromData(stateJSON)
	if err != nil {
		return nil
	}

	if shipState.V1 == nil || shipState.V1.Metadata == nil || shipState.V1.UpstreamContents == nil || shipState.V1.UpstreamContents.AppRelease == nil {
		return nil
	}

	license := shipstate.License{
		ID:  shipState.V1.Metadata.License.ID,
		Assignee: shipState.V1.Metadata.License.Assignee,
		Channel: shipState.V1.UpstreamContents.AppRelease.ChannelName,
		CreatedAt: shipState.V1.Metadata.License.CreatedAt,
		ExpiresAt: shipState.V1.Metadata.License.ExpiresAt,
		Type: shipState.V1.Metadata.License.Type,
		Entitlements: shipState.V1.UpstreamContents.AppRelease.Entitlements.Values,
	}

	licenseJSON, err := json.Marshal(license)
	if err != nil {
		return nil
	}

	return licenseJSON
}
