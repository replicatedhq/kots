package cli

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
