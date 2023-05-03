package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	if err := filepath.Walk(filepath.Join(workdir, "config", "crds"), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "failed to read crd file")
		}

		extensionsscheme.AddToScheme(scheme.Scheme)
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode(content, nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode crd")
		}

		crd, ok := obj.(*extensionsv1.CustomResourceDefinition)
		if !ok {
			return errors.New("failed to cast crd")
		}

		for _, version := range crd.Spec.Versions {
			outFile := fmt.Sprintf("%s-kots-%s.json", crd.Spec.Names.Singular, version.Name)
			if err := writeSchema(version.Schema.OpenAPIV3Schema, filepath.Join(workdir, v.GetString("output-dir"), outFile)); err != nil {
				return errors.Wrapf(err, "failed to write %s schema", crd.Spec.Names.Singular)
			}
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "failed to walk crds")
	}

	return nil
}

func writeSchema(schema *extensionsv1.JSONSchemaProps, outfile string) error {

	b, err := json.MarshalIndent(schema, "", "  ")
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
	replacer := strings.NewReplacer(
		`"type": "BoolString"`, `"oneOf": [{"type": "string"},{"type": "boolean"}]`,
		`"type": "QuotedBool"`, `"oneOf": [{"type": "string"},{"type": "boolean"}]`,
	)
	boolStringed := replacer.Replace(string(b))

	err = os.WriteFile(outfile, []byte(boolStringed), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}
