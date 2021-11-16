package base

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	rspb "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The following functions are partly copied from Helm code, but modified to better suit our usage (and also be available at all)

// newSecretsObject constructs a kubernetes Secret object
// to store a release. Each secret data entry is the base64
// encoded gzipped string of a release.
//
// The following labels are used within each secret:
//
//    "modifiedAt"    - timestamp indicating when this secret was last modified. (set in Update)
//    "createdAt"     - timestamp indicating when this secret was created. (set in Create)
//    "version"        - version of the release.
//    "status"         - status of the release (see pkg/release/status.go for variants)
//    "owner"          - owner of the secret, currently "helm".
//    "name"           - name of the release.
//
func newSecretsObject(rls *rspb.Release) (*v1.Secret, error) {
	const owner = "helm"
	key := makeKey(rls.Name, rls.Version)

	// encode the release
	s, err := encodeRelease(rls)
	if err != nil {
		return nil, err
	}

	lbs := map[string]string{
		//"createdAt": strconv.Itoa(int(time.Now().Unix())),
		"createdAt": strconv.Itoa(1), // make a constant so that there aren't spurious diffs
		"version":   strconv.Itoa(rls.Version),
		"status":    rls.Info.Status.String(),
		"owner":     owner,
		"name":      rls.Name,
	}

	// create and return secret object.
	// Helm 3 introduced setting the 'Type' field
	// in the Kubernetes storage object.
	// Helm defines the field content as follows:
	// <helm_domain>/<helm_object>.v<helm_object_version>
	// Type field for Helm 3: helm.sh/release.v1
	// Note: Version starts at 'v1' for Helm 3 and
	// should be incremented if the release object
	// metadata is modified.
	// This would potentially be a breaking change
	// and should only happen between major versions.
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   key,
			Labels: lbs,
		},
		Type: "helm.sh/release.v1",
		Data: map[string][]byte{"release": []byte(s)},
	}, nil
}

// encodeRelease encodes a release returning a base64 encoded
// gzipped string representation, or error.
func encodeRelease(rls *rspb.Release) (string, error) {
	b, err := json.Marshal(rls)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// makeKey concatenates the Kubernetes storage object type, a release name and version
// into a string with format:```<helm_storage_type>.<release_name>.v<release_version>```.
// The storage type is prepended to keep name uniqueness between different
// release storage types. An example of clash when not using the type:
// https://github.com/helm/helm/issues/6435.
// This key is used to uniquely identify storage objects.
func makeKey(rlsname string, version int) string {
	return fmt.Sprintf("%s.%s.v%d", storage.HelmStorageType, rlsname, version)
}
