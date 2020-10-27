package client

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/pkg/errors"
)

func (c *Client) patchResources(patchRequest PatchRequest) (*applyResult, error) {
	targetNamespace := c.TargetNamespace
	if patchRequest.Namespace != "." {
		targetNamespace = patchRequest.Namespace
	}

	kubernetesApplier, err := c.getApplier(patchRequest.KubectlVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(patchRequest.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode manifests")
	}

	byNamespace, err := docsByNamespace(decoded, targetNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get docs by requested namespace")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for requestedNamespace, mergedDocs := range byNamespace {
		if len(mergedDocs) == 0 {
			continue
		}

		log.Printf("patching manifest(s) in namespace %s", requestedNamespace)

		splitDocs := strings.Split(string(mergedDocs), "\n---\n")
		for _, doc := range splitDocs {
			if len(doc) == 0 {
				continue
			}

			for _, patch := range patchRequest.Patches {
				patchStdout, patchStderr, patchErr := kubernetesApplier.Patch(requestedNamespace, []byte(doc), patch)
				if patchErr != nil {
					log.Printf("stdout (patch) = %s", patchStdout)
					log.Printf("stderr (patch) = %s", patchStderr)
					log.Printf("error: %s", patchErr.Error())
					hasErr = true
				}
				if len(patchStdout) > 0 {
					multiStdout = append(multiStdout, patchStdout)
				}
				if len(patchStderr) > 0 {
					multiStderr = append(multiStderr, patchStderr)
				}
			}
		}
	}

	if len(multiStderr) == 0 {
		log.Printf("manifest(s) patched successfully")
	}

	result := &applyResult{ // TODO: change this
		hasErr:      hasErr,
		multiStderr: multiStderr,
		multiStdout: multiStdout,
	}
	return result, nil
}
