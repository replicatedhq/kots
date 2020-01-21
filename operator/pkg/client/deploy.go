package client

import (
	"bytes"
	"encoding/base64"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/util"
	"k8s.io/client-go/rest"
)

func (c *Client) diffAndRemovePreviousManifests(applicationManifests ApplicationManifests) error {
	decodedPrevious, err := base64.StdEncoding.DecodeString(applicationManifests.PreviousManifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode previous manifests")
	}

	decodedCurrent, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	// we need to find the gvk+names that are present in the previous, but not in the current and then remove them
	decodedPreviousStrings := strings.Split(string(decodedPrevious), "\n---\n")
	decodedPreviousMap := map[string]string{}
	for _, decodedPreviousString := range decodedPreviousStrings {
		decodedPreviousMap[GetGVKWithName([]byte(decodedPreviousString))] = decodedPreviousString
	}

	// now get the current names
	decodedCurrentStrings := strings.Split(string(decodedCurrent), "\n---\n")
	decodedCurrentMap := map[string]string{}
	for _, decodedCurrentString := range decodedCurrentStrings {
		decodedCurrentMap[GetGVKWithName([]byte(decodedCurrentString))] = decodedCurrentString
	}

	// now remove anything that's in previous but not in current
	kubectl, err := util.FindKubectlVersion(applicationManifests.KubectlVersion)
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := applier.NewKubectl(kubectl, "", "", config)
	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	for k, oldContents := range decodedPreviousMap {
		if _, ok := decodedCurrentMap[k]; !ok {
			gv, k, n, err := ParseSimpleGVK([]byte(oldContents))
			if err != nil {
				log.Printf("deleting unidentified manifest. unable to parse error: %s", err.Error())
			} else {
				log.Printf("deleting manifest(s): %s/%s/%s", gv, k, n)
			}
			stdout, stderr, err := kubernetesApplier.Remove(targetNamespace, []byte(oldContents), applicationManifests.Wait)
			if err != nil {
				log.Printf("stdout (delete) = %s", stdout)
				log.Printf("stderr (delete) = %s", stderr)
				log.Printf("error: %s", err.Error())
			} else {
				log.Printf("manifest(s) deleted: %s/%s/%s", gv, k, n)
			}
		}
	}

	return nil
}

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
			if len(docs) == 0 {
				continue
			}

			log.Printf("dry run applying manifests(s) in requested namespace: %s", requestedNamespace)
			dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(requestedNamespace, docs, true, applicationManifests.Wait)
			if dryRunErr != nil {
				log.Printf("stdout (dryrun) = %s", dryrunStdout)
				log.Printf("stderr (dryrun) = %s", dryrunStderr)
				log.Printf("error: %s", dryRunErr.Error())
			} else {
				log.Printf("dry run applied manifests(s) in requested namespace: %s", requestedNamespace)
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

		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply("", customResourceDefinitions, false, applicationManifests.Wait)
		if applyErr != nil {
			log.Printf("stdout (apply CRDS) = %s", applyStdout)
			log.Printf("stderr (apply CRDS) = %s", applyStderr)
			log.Printf("error (CRDS): %s", applyErr.Error())

			if err := c.sendResult(applicationManifests, applyErr != nil, []byte{}, []byte{}, applyStdout, applyStderr); err != nil {
				return errors.Wrap(err, "failed to report crd status")
			}

			return nil
		} else {
			log.Println("custom resource definition(s) applied")
		}

		// Give the API server a minute (well, 5 seconds) to cache the CRDs
		time.Sleep(time.Second * 5)
	}

	byNamespace, err := docsByNamespace(otherDocs, targetNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get docs by requested namespace")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for requestedNamespace, docs := range byNamespace {
		if len(docs) == 0 {
			continue
		}

		log.Printf("applying manifest(s) in namespace %s", requestedNamespace)
		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply(requestedNamespace, docs, false, applicationManifests.Wait)
		if applyErr != nil {
			log.Printf("stdout (apply) = %s", applyStdout)
			log.Printf("stderr (apply) = %s", applyStderr)
			log.Printf("error: %s", applyErr.Error())
			hasErr = true
		} else {
			log.Printf("manifest(s) applied in namespace %s", requestedNamespace)
		}
		if len(applyStdout) > 0 {
			multiStdout = append(multiStdout, applyStdout)
		}
		if len(applyStderr) > 0 {
			multiStderr = append(multiStderr, applyStderr)
		}
	}

	if err := c.sendResult(
		applicationManifests, hasErr, []byte{}, []byte{},
		bytes.Join(multiStdout, []byte("\n")), bytes.Join(multiStderr, []byte("\n")),
	); err != nil {
		return errors.Wrap(err, "failed to report status")
	}

	return nil
}
