// NOTE: This is a prototype for a command that will execute a Helm post-renderer.
package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func PostRendererCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "post-renderer",
		Short:         "execute a Helm post-renderer",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			license, err := getLicense(v)
			if err != nil {
				return errors.Wrap(err, "failed to get license")
			}

			if license == nil {
				return errors.New("no license found")
			}

			namespace := v.GetString("namespace")
			if namespace == "" {
				namespace = "default"
			}

			input, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("error reading input: %s", err)
			}
			// TODO: if input is empty, read from args

			workDir, err := os.MkdirTemp("", "kots-post-renderer")
			if err != nil {
				return fmt.Errorf("error creating temp dir: %s", err)
			}
			defer os.RemoveAll(workDir)

			// create tmpDir/base
			if err := os.Mkdir(fmt.Sprintf("%s/base", workDir), 0755); err != nil {
				return fmt.Errorf("error creating base dir: %s", err)
			}

			// create tmpDir/midstream
			if err := os.Mkdir(fmt.Sprintf("%s/midstream", workDir), 0755); err != nil {
				return fmt.Errorf("error creating midstream dir: %s", err)
			}

			// write the input to tmpDir/base/all.yaml
			if err := os.WriteFile(fmt.Sprintf("%s/base/all.yaml", workDir), input, 0644); err != nil {
				return fmt.Errorf("error writing file: %s", err)
			}

			// create base kustomization.yaml
			baseKustomization := kustomizetypes.Kustomization{
				TypeMeta: kustomizetypes.TypeMeta{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
				},
				Resources: []string{
					"all.yaml",
				},
			}

			baseKustomizationBytes, err := json.Marshal(baseKustomization)
			if err != nil {
				return fmt.Errorf("failed to marshal base kustomization: %v", err)
			}

			if err := os.WriteFile(fmt.Sprintf("%s/base/kustomization.yaml", workDir), baseKustomizationBytes, 0644); err != nil {
				return fmt.Errorf("error writing file: %s", err)
			}

			// split the input multidoc yaml into individual files
			splitDocs := util.ConvertToSingleDocs(input)

			docsWithImages := []k8sdoc.K8sDoc{}
			for _, doc := range splitDocs {
				parsed, err := k8sdoc.ParseYAML(doc)
				if err != nil {
					continue
				}

				images := parsed.ListImages()
				if len(images) > 0 {
					docsWithImages = append(docsWithImages, parsed)
				}
			}

			midstreamKustomization := kustomizetypes.Kustomization{
				TypeMeta: kustomizetypes.TypeMeta{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
				},
				Bases:                 []string{},
				Resources:             []string{},
				Patches:               []kustomizetypes.Patch{},
				PatchesStrategicMerge: []kustomizetypes.PatchStrategicMerge{},
			}

			dockercfgAuth := registry.DockercfgAuth{
				Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID))),
			}

			dockerCfgJSON := registry.DockerCfgJSON{
				Auths: map[string]registry.DockercfgAuth{},
			}

			registries := []string{
				"registry.replicated.com",
				"proxy.replicated.com",
			}

			for _, r := range registries {
				// we can get "host/namespace" here, which can break parts of kots that use hostname to lookup secret.
				host := strings.Split(r, "/")[0]
				dockerCfgJSON.Auths[host] = dockercfgAuth
			}

			secretData, err := json.Marshal(dockerCfgJSON)
			if err != nil {
				return fmt.Errorf("failed to marshal docker config: %v", err)
			}

			m := midstream.Midstream{
				Kustomization: &midstreamKustomization,
				AppPullSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-registry", license.Spec.AppSlug),
						Namespace: namespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": secretData,
					},
				},
				DocForPatches: docsWithImages,
			}

			options := midstream.WriteOptions{
				MidstreamDir: fmt.Sprintf("%s/midstream", workDir),
				BaseDir:      fmt.Sprintf("%s/base", workDir),
			}

			secretFilename, err := m.WritePullSecret(options)
			if err != nil {
				return errors.Wrap(err, "failed to write secret")
			}

			if secretFilename != "" {
				m.Kustomization.Resources = append(m.Kustomization.Resources, secretFilename)
			}

			if err := m.WriteObjectsWithPullSecret(options); err != nil {
				return errors.Wrap(err, "failed to write patches")
			}

			if err := m.WriteKustomization(options); err != nil {
				return errors.Wrap(err, "failed to write kustomization")
			}

			// run kustomize build
			kustomizeCmd := exec.Command("kustomize", "build", fmt.Sprintf("%s/midstream", workDir))
			kustomizeCmd.Stderr = os.Stderr
			kustomizeOutput, err := kustomizeCmd.Output()
			if err != nil {
				return fmt.Errorf("error running kustomize: %s", err)
			}

			fmt.Fprint(os.Stdout, string(kustomizeOutput))

			return nil
		},
	}

	cmd.Flags().String("license-file", "", "path to license file")
	cmd.Flags().StringP("namespace", "n", "", "namespace to use")

	return cmd
}
