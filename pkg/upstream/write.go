package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func WriteUpstream(u *types.Upstream, options types.WriteOptions) error {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	renderDir = path.Join(renderDir, "upstream")

	if options.IncludeAdminConsole {
		adminConsoleFiles, err := GenerateAdminConsoleFiles(renderDir, options)
		if err != nil {
			return errors.Wrap(err, "failed to generate admin console")
		}

		u.Files = append(u.Files, adminConsoleFiles...)
	}

	var previousInstallationContent []byte
	_, err := os.Stat(renderDir)
	if err == nil {
		_, err = os.Stat(path.Join(renderDir, "userdata", "installation.yaml"))
		if err == nil {
			c, err := ioutil.ReadFile(path.Join(renderDir, "userdata", "installation.yaml"))
			if err != nil {
				return errors.Wrap(err, "failed to read existing installation")
			}

			previousInstallationContent = c
		}

		if err := os.RemoveAll(renderDir); err != nil {
			return errors.Wrap(err, "failed to remove previous content in upstream")
		}
	}

	var prevInstallation *kotsv1beta1.Installation
	if previousInstallationContent != nil {
		decode := scheme.Codecs.UniversalDeserializer().Decode

		prevObj, _, err := decode(previousInstallationContent, nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode previous installation")
		}
		prevInstallation = prevObj.(*kotsv1beta1.Installation)
	}

	encryptionKey, err := getEncryptionKey(prevInstallation)
	if err != nil {
		return errors.Wrap(err, "failed to get encryption key")
	}
	_ = crypto.InitFromString(encryptionKey)

	for i, file := range u.Files {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if options.EncryptConfig {
			configValues := contentToConfigValues(file.Content)
			if configValues != nil {
				content, err := encryptConfigValues(configValues)
				if err != nil {
					return errors.Wrap(err, "failed to encrypt config values")
				}
				file.Content = content
				u.Files[i] = file
			}
		}

		identityConfig := contentToIdentityConfig(file.Content)
		if identityConfig != nil {
			content, err := maybeEncryptIdentityConfig(identityConfig)
			if err != nil {
				return errors.Wrap(err, "failed to encrypt identity config")
			}
			file.Content = content
			u.Files[i] = file
		}

		if archives.IsTGZ(file.Content) {
			// this is a helm chart, so we need to check if it is or contains the replicated-sdk
			reader := bytes.NewReader(file.Content)
			replicatedSDKChartName, isReplicatedSDK, err := FindReplicatedSDKChart(reader)
			if err != nil {
				return errors.Wrap(err, "failed to find replicated-sdk subchart")
			}

			if replicatedSDKChartName != "" {
				updatedContent, err := configureReplicatedSDK(file.Content, u, replicatedSDKChartName, isReplicatedSDK)
				if err != nil {
					return errors.Wrap(err, "failed to configure replicated sdk")
				}
				u.Files[i].Content = updatedContent
			}
		}

		if err := os.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write upstream file")
		}
	}

	channelID, channelName := "", ""
	if prevInstallation != nil && options.PreserveInstallation {
		channelID = prevInstallation.Spec.ChannelID
		channelName = prevInstallation.Spec.ChannelName
	} else {
		channelID = u.ChannelID
		channelName = u.ChannelName
	}

	installation := kotsv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: u.Name,
		},
		Spec: kotsv1beta1.InstallationSpec{
			UpdateCursor:             u.UpdateCursor,
			ChannelID:                channelID,
			ChannelName:              channelName,
			ReleaseSequence:          u.ReleaseSequence,
			VersionLabel:             u.VersionLabel,
			IsRequired:               u.IsRequired,
			ReleaseNotes:             u.ReleaseNotes,
			ReplicatedRegistryDomain: u.ReplicatedRegistryDomain,
			ReplicatedProxyDomain:    u.ReplicatedProxyDomain,
			EncryptionKey:            encryptionKey,
		},
	}

	if u.ReleasedAt != nil {
		releasedAt := metav1.NewTime(*u.ReleasedAt)
		installation.Spec.ReleasedAt = &releasedAt
	}

	if _, err := os.Stat(path.Join(renderDir, "userdata")); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(renderDir, "userdata"), 0755); err != nil {
			return errors.Wrap(err, "failed to create userdata dir")
		}
	}

	installationBytes := kotsutil.MustMarshalInstallation(&installation)
	err = ioutil.WriteFile(path.Join(renderDir, "userdata", "installation.yaml"), installationBytes, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}

	return nil
}

func getEncryptionKey(prevInstallation *kotsv1beta1.Installation) (string, error) {
	if prevInstallation == nil {
		return "", nil
	}

	return prevInstallation.Spec.EncryptionKey, nil
}

func encryptConfigValues(configValues *kotsv1beta1.ConfigValues) ([]byte, error) {
	for k, v := range configValues.Spec.Values {
		if v.ValuePlaintext == "" {
			continue
		}

		v.Value = base64.StdEncoding.EncodeToString(crypto.Encrypt([]byte(v.ValuePlaintext)))
		v.ValuePlaintext = ""

		configValues.Spec.Values[k] = v
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode config values")
	}

	return b.Bytes(), nil
}

func maybeEncryptIdentityConfig(identityConfig *kotsv1beta1.IdentityConfig) ([]byte, error) {
	identityConfig.Spec.ClientSecret.EncryptValue()
	identityConfig.Spec.DexConnectors.EncryptValue()

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(identityConfig, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode identity config")
	}

	return b.Bytes(), nil
}

// FindReplicatedSDKChart will look for a chart with the name "replicated" or "replicated-sdk" in the archive
// and return the name of the chart and whether it is the parent chart or a subchart
func FindReplicatedSDKChart(archive io.Reader) (string, bool, error) {
	gzReader, err := gzip.NewReader(archive)
	if err != nil {
		return "", false, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, errors.Wrap(err, "failed to read header from tar")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if filepath.Base(header.Name) == "Chart.yaml" {
				content, err := io.ReadAll(tarReader)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to read file %s", header.Name)
				}

				chartInfo := struct {
					ChartName string `json:"name" yaml:"name"`
				}{}
				if err := yaml.Unmarshal(content, &chartInfo); err != nil {
					return "", false, errors.Wrapf(err, "failed to unmarshal %s", header.Name)
				}

				// TEMPORARY: for backwards compatibility, check for "replicated-sdk" name as well
				if chartInfo.ChartName == "replicated" || chartInfo.ChartName == "replicated-sdk" {
					// check if the sdk is the parent chart or a subchart based on the path
					replicatedSDKChartName := chartInfo.ChartName
					isReplicatedSDK := !strings.Contains(filepath.Dir(header.Name), string(os.PathSeparator))
					return replicatedSDKChartName, isReplicatedSDK, nil
				}
			}
		}
	}

	return "", false, nil
}

func configureReplicatedSDK(chartContent []byte, u *types.Upstream, replicatedSDKChartName string, isReplicatedSDK bool) ([]byte, error) {
	reader := bytes.NewReader(chartContent)
	unrenderedContents, pathInArchive, extractedArchiveRoot, err := findTopLevelChartValues(reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find top level chart values")
	}
	defer os.RemoveAll(extractedArchiveRoot)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	renderedValuesContents, err := renderValuesYAMLForLicense(clientset, store.GetStore(), unrenderedContents, u, replicatedSDKChartName, isReplicatedSDK)
	if err != nil {
		return nil, errors.Wrap(err, "render values.yaml")
	}

	if err := os.WriteFile(filepath.Join(extractedArchiveRoot, pathInArchive), renderedValuesContents, 0644); err != nil {
		return nil, errors.Wrap(err, "write rendered values.yaml")
	}

	updatedArchive, err := packageChartArchive(extractedArchiveRoot)
	if err != nil {
		return nil, errors.Wrap(err, "package chart archive")
	}
	defer os.RemoveAll(updatedArchive)

	renderedContents, err := os.ReadFile(updatedArchive)
	if err != nil {
		return nil, errors.Wrap(err, "read updated archive")
	}

	return renderedContents, nil
}

func findTopLevelChartValues(r io.Reader) (valuesYaml []byte, pathInArchive string, workspace string, finalErr error) {
	workspace, err := os.MkdirTemp("", "extracted-chart-")
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create temp directory")
		return
	}

	defer func() {
		if finalErr != nil {
			os.RemoveAll(workspace)
			workspace = ""
		}
	}()

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create gzip reader")
		return
	}

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			finalErr = errors.Wrap(err, "failed to read header from tar")
			return
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(workspace, header.Name), fs.FileMode(header.Mode)); err != nil {
				finalErr = errors.Wrap(err, "failed to create directory from archive")
				return
			}
		case tar.TypeReg:
			content, err := io.ReadAll(tarReader)
			if err != nil {
				finalErr = errors.Wrap(err, "failed to read file")
				return
			}

			if filepath.Base(header.Name) == "values.yaml" {
				// only get the values.yaml from the top level chart
				p := filepath.Dir(header.Name)
				if !strings.Contains(p, string(os.PathSeparator)) {
					pathInArchive = header.Name
					valuesYaml = content
				}
			}

			dir := filepath.Dir(filepath.Join(workspace, header.Name))
			if err := os.MkdirAll(dir, 0700); err != nil {
				finalErr = errors.Wrap(err, "failed to create directory from filename")
				return
			}

			outFile, err := os.Create(filepath.Join(workspace, header.Name))
			if err != nil {
				finalErr = errors.Wrap(err, "failed to create file")
				return
			}
			defer outFile.Close()
			if err := os.WriteFile(outFile.Name(), content, header.FileInfo().Mode()); err != nil {
				finalErr = errors.Wrap(err, "failed to write file")
				return
			}
		}
	}

	return
}

type LicenseField struct {
	Name        string                `json:"name" yaml:"name"`
	Title       string                `json:"title" yaml:"title"`
	Description string                `json:"description" yaml:"description"`
	Value       interface{}           `json:"value" yaml:"value"`
	ValueType   string                `json:"valueType" yaml:"valueType"`
	HelmPath    *string               `json:"helmPath,omitempty" yaml:"helmPath,omitempty"`
	IsHidden    bool                  `json:"isHidden,omitempty" yaml:"isHidden,omitempty"`
	Signature   LicenseFieldSignature `json:"signature,omitempty" yaml:"signature,omitempty"`
}

type LicenseFieldSignature struct {
	V1 string `json:"v1,omitempty" yaml:"v1,omitempty"` // this is a base64 encoded string because yaml.Unmarshal doesn't automatically convert base64 to []byte like json.Unmarshal does
}

func renderValuesYAMLForLicense(clientset kubernetes.Interface, kotsStore store.Store, unrenderedContents []byte, u *types.Upstream, replicatedSDKChartName string, isReplicatedSDK bool) ([]byte, error) {
	var licenseBytes []byte
	var licenseFields map[string]LicenseField
	var clusterID string
	var appID string
	var replicatedAppEndpoint string
	var dockerCfgJson string
	if u.License != nil {
		licenseBytes = MustMarshalLicense(u.License)
		licenseFields = map[string]LicenseField{}
		for k, v := range u.License.Spec.Entitlements {
			// TODO: support individual license field signatures
			licenseFields[k] = LicenseField{
				Name:        k,
				Title:       v.Title,
				Description: v.Description,
				Value:       v.Value.Value(),
				ValueType:   v.ValueType,
			}
		}
		appSlug := u.License.Spec.AppSlug
		replicatedAppEndpoint = u.License.Spec.Endpoint

		auth := fmt.Sprintf("%s:%s", u.License.Spec.LicenseID, u.License.Spec.LicenseID)
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		dockercfg := map[string]interface{}{
			"auths": map[string]interface{}{
				u.ReplicatedProxyDomain: map[string]string{
					"auth": encodedAuth,
				},
				u.ReplicatedRegistryDomain: map[string]string{
					"auth": encodedAuth,
				},
			},
		}

		b, err := json.Marshal(dockercfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal dockercfg")
		}

		dockerCfgJson = base64.StdEncoding.EncodeToString(b)

		clusterID = k8sutil.GetKotsadmClusterID(clientset)
		appID, err = kotsStore.GetAppIDFromSlug(appSlug)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app id from slug")
		}
	}

	var appName string
	var statusInformers []string
	if u.Application != nil {
		appName = u.Application.Spec.Title
		statusInformers = u.Application.Spec.StatusInformers
	}

	opts := ReplicatedValuesOptions{
		LicenseYAML:           licenseBytes,
		ReleaseSequence:       u.ReleaseSequence,
		ReleaseCreatedAt:      u.ReleasedAt,
		ReleaseNotes:          u.ReleaseNotes,
		VersionLabel:          u.VersionLabel,
		ClusterID:             clusterID,
		AppID:                 appID,
		AppName:               appName,
		ChannelID:             u.ChannelID,
		ChannelName:           u.ChannelName,
		ReplicatedAppEndpoint: replicatedAppEndpoint,
		StatusInformers:       statusInformers,
	}

	if channelSequence, err := strconv.ParseInt(u.UpdateCursor, 10, 64); err == nil {
		opts.ChannelSequence = channelSequence
	}

	var valuesNodes yaml.Node
	if err := yaml.Unmarshal([]byte(unrenderedContents), &valuesNodes); err != nil {
		return nil, errors.Wrap(err, "unmarshal values")
	}

	if len(valuesNodes.Content) == 0 {
		return nil, errors.New("no content")
	}

	if isReplicatedSDK {
		err := addReplicatedValuesForSDK(valuesNodes.Content[0], opts, "", true)
		if err != nil {
			return nil, errors.Wrap(err, "add values for the replicated sdk chart")
		}
	} else if replicatedSDKChartName != "" {
		// replicated sdk is included as a subchart.
		// make sure to add the values under the subchart name
		// as helm expects the field name to match the subchart name
		err := addReplicatedValuesForSDK(valuesNodes.Content[0], opts, replicatedSDKChartName, false)
		if err != nil {
			return nil, errors.Wrap(err, "add values for the replicated sdk subchart")
		}
	}

	globalOpts := ReplicatedGlobalValuesOptions{
		ChannelName:      u.ChannelName,
		CustomerEmail:    u.License.Spec.CustomerEmail,
		CustomerName:     u.License.Spec.CustomerName,
		DockerConfigJSON: dockerCfgJson,
		LicenseID:        u.License.Spec.LicenseID,
		LicenseType:      u.License.Spec.LicenseType,
		LicenseFields:    licenseFields,
	}

	err := addReplicatedGlobalValues(valuesNodes.Content[0], globalOpts)
	if err != nil {
		return nil, errors.Wrap(err, "add replicated global values")
	}

	licenseFieldNodes, err := convertLicenseFieldsToYamlNodes(licenseFields)
	if err != nil {
		return nil, errors.Wrap(err, "convert license fields to yaml nodes")
	}

	newValues := kotsutil.ContentToDocNode(&valuesNodes, kotsutil.MergeYAMLNodes(valuesNodes.Content, licenseFieldNodes))

	renderedContents, err := kotsutil.NodeToYAML(newValues)
	if err != nil {
		return nil, errors.Wrap(err, "render values")
	}

	return renderedContents, nil
}

type ReplicatedValuesOptions struct {
	LicenseYAML           []byte
	ChannelSequence       int64
	ReleaseSequence       int64
	ReleaseCreatedAt      *time.Time
	ReleaseNotes          string
	VersionLabel          string
	ClusterID             string
	AppID                 string
	AppName               string
	ChannelID             string
	ChannelName           string
	ReplicatedAppEndpoint string
	StatusInformers       []string
}

func addReplicatedValuesForSDK(doc *yaml.Node, opts ReplicatedValuesOptions, replicatedSDKSubchartName string, isReplicatedSDK bool) error {
	replicatedValues := map[string]interface{}{
		"license":               string(opts.LicenseYAML),
		"channelSequence":       opts.ChannelSequence,
		"releaseSequence":       opts.ReleaseSequence,
		"releaseNotes":          opts.ReleaseNotes,
		"versionLabel":          opts.VersionLabel,
		"replicatedAppEndpoint": opts.ReplicatedAppEndpoint,
		"appName":               opts.AppName,
		"channelID":             opts.ChannelID,
		"channelName":           opts.ChannelName,
		"replicatedID":          opts.ClusterID,
		"appID":                 opts.AppID,
		"userAgent":             buildversion.GetUserAgent(),
		"statusInformers":       opts.StatusInformers,
	}

	if opts.ReleaseCreatedAt != nil {
		replicatedValues["releaseCreatedAt"] = opts.ReleaseCreatedAt.Format(time.RFC3339)
	}

	targetNode := doc
	hasReplicatedValues := false

	if replicatedSDKSubchartName != "" {
		for i, n := range doc.Content {
			if n.Value == replicatedSDKSubchartName {
				targetNode = doc.Content[i+1]
				hasReplicatedValues = true
				break
			}
		}
	}

	v := replicatedValues
	if replicatedSDKSubchartName != "" && !hasReplicatedValues {
		v = map[string]interface{}{
			replicatedSDKSubchartName: replicatedValues,
		}
	}

	additionalYAML, err := yaml.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal additional values")
	}

	var additionalNode yaml.Node
	if err := yaml.Unmarshal([]byte(additionalYAML), &additionalNode); err != nil {
		return errors.Wrap(err, "unmarshal additional values")
	}

	if hasReplicatedValues || isReplicatedSDK {
		targetNode.Content = kotsutil.MergeYAMLNodes(targetNode.Content, additionalNode.Content[0].Content)
	} else {
		targetNode.Content = append(targetNode.Content, additionalNode.Content[0].Content...)
	}

	return nil
}

type ReplicatedGlobalValuesOptions struct {
	ChannelName      string
	CustomerEmail    string
	CustomerName     string
	DockerConfigJSON string
	LicenseID        string
	LicenseType      string
	LicenseFields    map[string]LicenseField
}

func addReplicatedGlobalValues(doc *yaml.Node, opts ReplicatedGlobalValuesOptions) error {
	targetNode := doc
	hasGlobal := false
	for i, n := range doc.Content {
		if n.Value == "global" {
			targetNode = doc.Content[i+1]
			hasGlobal = true
			break
		}
	}

	hasGlobalReplicated := false
	if hasGlobal {
		for i, n := range targetNode.Content {
			if n.Value == "replicated" {
				targetNode = targetNode.Content[i+1]
				hasGlobalReplicated = true
				break
			}
		}
	}

	replicatedValues := map[string]interface{}{
		"channelName":      opts.ChannelName,
		"customerName":     opts.CustomerName,
		"customerEmail":    opts.CustomerEmail,
		"licenseID":        opts.LicenseID,
		"licenseType":      opts.LicenseType,
		"dockerconfigjson": opts.DockerConfigJSON,
		"licenseFields":    opts.LicenseFields,
	}

	v := replicatedValues
	if !hasGlobalReplicated {
		v = map[string]interface{}{
			"replicated": v,
		}
		if !hasGlobal {
			v = map[string]interface{}{
				"global": v,
			}
		}
	}

	additionalYAML, err := yaml.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal additional values")
	}

	var additionalNode yaml.Node
	if err := yaml.Unmarshal([]byte(additionalYAML), &additionalNode); err != nil {
		return errors.Wrap(err, "unmarshal additional values")
	}

	if hasGlobalReplicated || hasGlobal {
		targetNode.Content = kotsutil.MergeYAMLNodes(targetNode.Content, additionalNode.Content[0].Content)
	} else {
		targetNode.Content = append(targetNode.Content, additionalNode.Content[0].Content...)
	}

	return nil
}

func convertLicenseFieldsToYamlNodes(licenseFields map[string]LicenseField) ([]*yaml.Node, error) {
	nestedFields := map[string]interface{}{}
	for fieldName, field := range licenseFields {
		if field.HelmPath == nil || *field.HelmPath == "" {
			// Not all license fields have a helm path
			continue
		}

		// Skip ".Values." prefix if it exists
		pathParts := strings.Split(*field.HelmPath, ".")
		if len(pathParts) < 2 {
			return nil, errors.Errorf("field %s has invalid helm path %q", fieldName, *field.HelmPath)
		}
		if pathParts[0] == "" {
			pathParts = pathParts[1:]
		}
		if pathParts[0] == "Values" {
			pathParts = pathParts[1:]
		}

		if len(pathParts) == 1 {
			nestedFields[pathParts[0]] = field.Value
			continue
		}

		var nextMap map[string]interface{}
		if m, ok := nestedFields[pathParts[0]]; ok {
			nextMap = m.(map[string]interface{})
		} else {
			nextMap = map[string]interface{}{}
		}

		nestedFields[pathParts[0]] = nextMap
		for i := 1; i < len(pathParts)-1; i++ {
			var newNextMap map[string]interface{}
			if m, ok := nextMap[pathParts[i]]; ok {
				newNextMap = m.(map[string]interface{})
			} else {
				newNextMap = map[string]interface{}{}
				nextMap[pathParts[i]] = newNextMap
			}
			nextMap = newNextMap
		}
		nextMap[pathParts[len(pathParts)-1]] = field.Value
	}

	valuesYaml, err := yaml.Marshal(nestedFields)
	if err != nil {
		return nil, errors.Wrap(err, "marshal values")
	}

	var v yaml.Node
	if err := yaml.Unmarshal([]byte(valuesYaml), &v); err != nil {
		return nil, errors.Wrap(err, "unmarshal values")
	}

	return v.Content, nil
}

func packageChartArchive(extractedArchiveRoot string) (string, error) {
	renderedPath, err := ioutil.TempFile("", "rendered-chart-")
	if err != nil {
		return "", errors.Wrap(err, "create temp file")
	}

	file, err := os.Create(renderedPath.Name())
	if err != nil {
		return "", errors.Wrap(err, "create file")
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(extractedArchiveRoot, func(path string, info os.FileInfo, err error) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			return err
		}

		if fi.Mode().IsRegular() {
			if err := addFileToTarWriter(extractedArchiveRoot, path, tarWriter); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return "", errors.Wrap(err, "walk file tree")
	}

	return renderedPath.Name(), nil
}

func addFileToTarWriter(basePath string, filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not open file '%s', got error '%s'", filePath, err.Error()))
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return errors.New(fmt.Sprintf("Could not get stat for file '%s', got error '%s'", filePath, err.Error()))
	}

	rel, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not get relative path for file '%s', got error '%s'", filePath, err.Error()))
	}

	header := &tar.Header{
		Name:    rel,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not write header for file '%s', got error '%s'", filePath, err.Error()))
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not copy the file '%s' data to the tarball, got error '%s'", filePath, err.Error()))
	}

	return nil
}
