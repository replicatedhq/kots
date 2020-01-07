package client

import (
	"encoding/base64"
	"log"
	"time"

	"github.com/pkg/errors"
)

func (c *Client) ensureResourcesPresent(applicationManifests ApplicationManifests) error {
	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	kubernetesApplier, err := c.getApplier(applicationManifests.KubectlVersion)
	if err != nil {
		return errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	customResourceDefinitions, otherDocs, err := splitMutlidocYAMLIntoCRDsAndOthers(decoded)
	if err != nil {
		return errors.Wrap(err, "failed to split decoded into crds and other")
	}

	// We don't dry run if there's a crd becasue there's a likely chance that the
	// other docs has a custom resource using it
	shouldDryRun := customResourceDefinitions == nil
	if shouldDryRun {
		byNamespace, err := docsByNamespace(decoded, targetNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to get docs by requested namespace")
		}

		for requestedNamespace, docs := range byNamespace {
			log.Printf("dry run applying manifests(s) in requested namespace: %s\n", requestedNamespace)
			dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(requestedNamespace, docs, true)
			if dryRunErr != nil {
				log.Printf("stdout (dryrun) = %s", dryrunStdout)
				log.Printf("stderr (dryrun) = %s", dryrunStderr)
				log.Printf("error: %s", dryRunErr.Error())
			}

			if dryRunErr != nil {
				if err := c.sendResult(applicationManifests, true, dryrunStdout, dryrunStderr, []byte{}, []byte{}); err != nil {
					return errors.Wrap(err, "failed to report dry run status")
				}

				return nil // don't return an error because execution is proper, the api now has the error
			}
		}

	}

	if len(customResourceDefinitions) > 0 {
		log.Println("applying custom resource definition(s)")

		// CRDs don't have namespaces, so we can skip splitting

		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply("", customResourceDefinitions, false)
		if applyErr != nil {
			log.Printf("stdout (apply CRDS) = %s", applyStdout)
			log.Printf("stderr (apply CRDS) = %s", applyStderr)
			log.Printf("error (CRDS): %s", applyErr.Error())

			if err := c.sendResult(applicationManifests, applyErr != nil, []byte{}, []byte{}, applyStdout, applyStderr); err != nil {
				return errors.Wrap(err, "failed to report crd status")
			}

			return nil
		}

		// Give the API server a minute (well, 5 seconds) to cache the CRDs
		time.Sleep(time.Second * 5)
	}

	byNamespace, err := docsByNamespace(otherDocs, targetNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get docs by requested namespace")
	}

	for requestedNamespace, docs := range byNamespace {
		log.Printf("applying manifest(s) in namespace %s\n", requestedNamespace)
		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply(requestedNamespace, docs, false)
		if err != nil {
			log.Printf("stdout (apply) = %s", applyStdout)
			log.Printf("stderr (apply) = %s", applyStderr)
			log.Printf("error: %s", applyErr.Error())
		}

		if err := c.sendResult(applicationManifests, applyErr != nil, []byte{}, []byte{}, applyStdout, applyStderr); err != nil {
			return errors.Wrap(err, "failed to report status")
		}
	}

	return nil
}
