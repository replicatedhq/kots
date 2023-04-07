package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/postrenderer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func PostRendererCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "post-renderer",
		Short:         "execute a Helm post-renderer",
		Long:          `This command is used to execute a Helm post-renderer. It is not intended to be used directly.`,
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

			// write input to a *bytes.Buffer
			inBuffer := bytes.NewBuffer(input)

			kustomizeBinPath := v.GetString("kustomize-bin-path")
			if kustomizeBinPath == "" {
				// default to kustomize in PATH
				kustomizeBinPath, err = exec.LookPath("kustomize")
				if err != nil {
					return fmt.Errorf("error finding kustomize in PATH: %s", err)
				}
			}

			rootDir := v.GetString("root-dir")
			if rootDir == "" {
				// default to current working directory
				rootDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("error getting current working directory: %s", err)
				}
			}

			downstream := v.GetString("downstream")
			if downstream == "" {
				// default to "this-cluster"
				downstream = "this-cluster"
			}

			appSlug := v.GetString("app-slug")
			if appSlug == "" {
				return fmt.Errorf("app-slug is required")
			}

			kotsApplicationPath := v.GetString("kots-application")
			if kotsApplicationPath == "" {
				return fmt.Errorf("kots-application is required")
			}

			kotsApplicationContents, err := os.ReadFile(kotsApplicationPath)
			if err != nil {
				return fmt.Errorf("failed to read kots application: %v", err)
			}

			obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(kotsApplicationContents, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to decode kots application: %v", err)
			}

			kotsApplication, ok := obj.(*kotsv1beta1.Application)
			if !ok {
				return fmt.Errorf("failed to cast kots application")
			}

			postRenderer := postrenderer.NewPostRenderer(&postrenderer.PostRendererOptions{
				KustomizeBinPath:     kustomizeBinPath,
				RootKustomizationDir: rootDir,
				Downstream:           downstream,
				ReleaseName:          v.GetString("release-name"),
				Namespace:            namespace,
				AppSlug:              appSlug,
				RegistryHost:         v.GetString("registry-host"),
				RegistryUsername:     v.GetString("registry-username"),
				RegistryPassword:     v.GetString("registry-password"),
				License:              license,
				KotsApplication:      kotsApplication,
			})

			outBuffer, err := postRenderer.Run(inBuffer)
			if err != nil {
				return fmt.Errorf("error running post-renderer: %s", err)
			}

			if _, err = os.Stdout.Write(outBuffer.Bytes()); err != nil {
				return fmt.Errorf("error writing output: %s", err)
			}

			// workDir, err := os.MkdirTemp("", "kots-post-renderer")
			// if err != nil {
			// 	return fmt.Errorf("error creating temp dir: %s", err)
			// }
			// defer os.RemoveAll(workDir)

			// // create tmpDir/base
			// if err := os.Mkdir(fmt.Sprintf("%s/base", workDir), 0755); err != nil {
			// 	return fmt.Errorf("error creating base dir: %s", err)
			// }

			// // create tmpDir/midstream
			// if err := os.Mkdir(fmt.Sprintf("%s/midstream", workDir), 0755); err != nil {
			// 	return fmt.Errorf("error creating midstream dir: %s", err)
			// }

			// // write the input to tmpDir/base/all.yaml
			// if err := os.WriteFile(fmt.Sprintf("%s/base/all.yaml", workDir), input, 0644); err != nil {
			// 	return fmt.Errorf("error writing file: %s", err)
			// }

			// // create base kustomization.yaml
			// baseKustomization := kustomizetypes.Kustomization{
			// 	TypeMeta: kustomizetypes.TypeMeta{
			// 		APIVersion: "kustomize.config.k8s.io/v1beta1",
			// 		Kind:       "Kustomization",
			// 	},
			// 	Resources: []string{
			// 		"all.yaml",
			// 	},
			// }

			// baseKustomizationBytes, err := json.Marshal(baseKustomization)
			// if err != nil {
			// 	return fmt.Errorf("failed to marshal base kustomization: %v", err)
			// }

			// if err := os.WriteFile(fmt.Sprintf("%s/base/kustomization.yaml", workDir), baseKustomizationBytes, 0644); err != nil {
			// 	return fmt.Errorf("error writing file: %s", err)
			// }

			// // split the input multidoc yaml into individual files
			// splitDocs := util.ConvertToSingleDocs(input)

			// docsWithImages := []k8sdoc.K8sDoc{}
			// for _, doc := range splitDocs {
			// 	parsed, err := k8sdoc.ParseYAML(doc)
			// 	if err != nil {
			// 		continue
			// 	}

			// 	images := parsed.ListImages()
			// 	if len(images) > 0 {
			// 		docsWithImages = append(docsWithImages, parsed)
			// 	}
			// }

			// midstreamKustomization := kustomizetypes.Kustomization{
			// 	TypeMeta: kustomizetypes.TypeMeta{
			// 		APIVersion: "kustomize.config.k8s.io/v1beta1",
			// 		Kind:       "Kustomization",
			// 	},
			// 	Bases:                 []string{},
			// 	Resources:             []string{},
			// 	Patches:               []kustomizetypes.Patch{},
			// 	PatchesStrategicMerge: []kustomizetypes.PatchStrategicMerge{},
			// }

			// dockercfgAuth := registry.DockercfgAuth{
			// 	Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID))),
			// }

			// dockerCfgJSON := registry.DockerCfgJSON{
			// 	Auths: map[string]registry.DockercfgAuth{},
			// }

			// registries := []string{
			// 	"registry.replicated.com",
			// 	"proxy.replicated.com",
			// }

			// for _, r := range registries {
			// 	// we can get "host/namespace" here, which can break parts of kots that use hostname to lookup secret.
			// 	host := strings.Split(r, "/")[0]
			// 	dockerCfgJSON.Auths[host] = dockercfgAuth
			// }

			// secretData, err := json.Marshal(dockerCfgJSON)
			// if err != nil {
			// 	return fmt.Errorf("failed to marshal docker config: %v", err)
			// }

			// m := midstream.Midstream{
			// 	Kustomization: &midstreamKustomization,
			// 	AppPullSecret: &corev1.Secret{
			// 		TypeMeta: metav1.TypeMeta{
			// 			APIVersion: "v1",
			// 			Kind:       "Secret",
			// 		},
			// 		ObjectMeta: metav1.ObjectMeta{
			// 			Name:      fmt.Sprintf("%s-registry", license.Spec.AppSlug),
			// 			Namespace: namespace,
			// 		},
			// 		Type: corev1.SecretTypeDockerConfigJson,
			// 		Data: map[string][]byte{
			// 			".dockerconfigjson": secretData,
			// 		},
			// 	},
			// 	DocForPatches: docsWithImages,
			// }

			// options := midstream.WriteOptions{
			// 	MidstreamDir: fmt.Sprintf("%s/midstream", workDir),
			// 	BaseDir:      fmt.Sprintf("%s/base", workDir),
			// }

			// secretFilename, err := m.WritePullSecret(options)
			// if err != nil {
			// 	return errors.Wrap(err, "failed to write secret")
			// }

			// if secretFilename != "" {
			// 	m.Kustomization.Resources = append(m.Kustomization.Resources, secretFilename)
			// }

			// if err := m.WriteObjectsWithPullSecret(options); err != nil {
			// 	return errors.Wrap(err, "failed to write patches")
			// }

			// if err := m.WriteKustomization(options); err != nil {
			// 	return errors.Wrap(err, "failed to write kustomization")
			// }

			// // run kustomize build
			// kustomizeCmd := exec.Command("kustomize", "build", fmt.Sprintf("%s/midstream", workDir))
			// kustomizeCmd.Stderr = os.Stderr
			// kustomizeOutput, err := kustomizeCmd.Output()
			// if err != nil {
			// 	return fmt.Errorf("error running kustomize: %s", err)
			// }

			// fmt.Fprint(os.Stdout, string(kustomizeOutput))

			return nil
		},
	}

	cmd.Flags().String("kustomize-bin-path", "", "path to the kustomize binary (defaults to kustomize in PATH)")
	cmd.Flags().String("root-dir", "", "path to the root working directory for kustomize (defaults to current working directory)")
	cmd.Flags().String("downstream", "", "name of the downstream to use (defaults to 'this-cluster')")

	cmd.Flags().String("license-file", "", "path to license file (required)")
	cmd.Flags().String("release-name", "", "release name (required)")
	cmd.Flags().StringP("namespace", "n", "", "application target namespace (defaults to 'default')")
	cmd.Flags().String("app-slug", "", "application slug (required)")
	cmd.Flags().String("kots-application", "", "path to the kots application (required)")

	cmd.Flags().String("registry-hostname", "", "registry hostname (optional)")
	cmd.Flags().String("registry-username", "", "registry username (optional)")
	cmd.Flags().String("registry-password", "", "registry password (optional)")

	return cmd
}
