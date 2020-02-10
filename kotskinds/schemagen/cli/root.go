package cli

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "schemagen",
		Short:        "Generate openapischemas for the kinds in this project",
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			return generateSchemas(v)
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.Flags().String("output-dir", "./schemas", "directory to save the schemas in")

	viper.BindPFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("TROUBLESHOOT")
	viper.AutomaticEnv()
}

func generateSchemas(v *viper.Viper) error {
	// we generate schemas from the config/crds in the root of this project
	// those crds can be created from controller-gen or by running `make openapischema`

	workdir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "failed to get workdir")
	}

	airgapContent, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_airgaps.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read airgap crd")
	}
	if err := generateSchemaFromCRD(airgapContent, filepath.Join(workdir, v.GetString("output-dir"), "airgap-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write airgap schema")
	}

	applicationContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_applications.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read application crd")
	}
	if err := generateSchemaFromCRD(applicationContents, filepath.Join(workdir, v.GetString("output-dir"), "application-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write application schema")
	}

	configContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_configs.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read config crd")
	}
	if err := generateSchemaFromCRD(configContents, filepath.Join(workdir, v.GetString("output-dir"), "config-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write config schema")
	}

	configValuesContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_configvalues.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read configvalues crd")
	}
	if err := generateSchemaFromCRD(configValuesContents, filepath.Join(workdir, v.GetString("output-dir"), "configvalues-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write configvalues schema")
	}

	helmChartContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_helmcharts.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read helmchart crd")
	}
	if err := generateSchemaFromCRD(helmChartContents, filepath.Join(workdir, v.GetString("output-dir"), "helmchart-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write helmchart schema")
	}

	installationContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_installations.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read installations crd")
	}
	if err := generateSchemaFromCRD(installationContents, filepath.Join(workdir, v.GetString("output-dir"), "installation-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write installation schema")
	}

	licenseContents, err := ioutil.ReadFile(filepath.Join(workdir, "config", "crds", "kots.io_licenses.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read license crd")
	}
	if err := generateSchemaFromCRD(licenseContents, filepath.Join(workdir, v.GetString("output-dir"), "license-kots-v1beta1.json")); err != nil {
		return errors.Wrap(err, "failed to write license schema")
	}

	return nil
}

func generateSchemaFromCRD(crd []byte, outfile string) error {
	extensionsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(crd, nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to decode crd")
	}

	customResourceDefinition := obj.(*extensionsv1beta1.CustomResourceDefinition)

	b, err := json.MarshalIndent(customResourceDefinition.Spec.Validation.OpenAPIV3Schema, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}

	_, err = os.Stat(outfile)
	if err == nil {
		if err := os.Remove(outfile); err != nil {
			return errors.Wrap(err, "failed to remove file")
		}
	}

	d, _ := path.Split(outfile)
	_, err = os.Stat(d)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(d, 0755); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	// whoa now
	// working around the fact that controller-gen doesn't have tags to generate oneOf schemas, so this is hacky.
	// going to work to add an issue there to support and if they accept, this terrible thing can go away
	boolStringed := strings.ReplaceAll(string(b), `"type": "BoolString"`, `"oneOf": [{"type": "string"},{"type": "boolean"}]`)

	err = ioutil.WriteFile(outfile, []byte(boolStringed), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}
