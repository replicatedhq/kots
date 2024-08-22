package cli

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/util"
)

func ExpandDir(input string) string {
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "~") {
		input = filepath.Join(util.HomeDir(), strings.TrimPrefix(input, "~"))
	}

	uploadPath, err := filepath.Abs(input)
	if err != nil {
		panic(errors.Wrapf(err, "unable to expand %q to absolute path", input))
	}

	return uploadPath
}

func getHostFromEndpoint(endpoint string) (string, error) {
	if !strings.HasPrefix(endpoint, "http") {
		// url.Parse doesn't work without scheme
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse endpoint")
	}

	return parsed.Host, nil
}

func splitEndpointAndNamespace(endpoint string) (string, string) {
	registryEndpoint := endpoint
	registryNamespace := ""
	parts := strings.Split(endpoint, "/")
	if len(parts) > 1 {
		registryEndpoint = parts[0]
		registryNamespace = strings.Join(parts[1:], "/")
	}
	return registryEndpoint, registryNamespace
}

func extractPreferredChannelSlug(log *logger.CLILogger, upstreamURI string) (string, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse uri")
	}

	replicatedUpstream, err := replicatedapp.ParseReplicatedURL(u)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse replicated url")
	}

	if replicatedUpstream.Channel != nil {
		return *replicatedUpstream.Channel, nil
	}

	if log != nil {
		log.ActionWithoutSpinner("No channel specified in upstream URI, falling back to channel slug 'stable.")
	}
	return "stable", nil
}
