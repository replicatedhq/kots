package kotsutil_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	kurlv1beta1 "github.com/replicatedhq/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
)

var _ = Describe("Kots", func() {
	Describe("GetKotsKindsPath()", func() {
		It("returns the path to the kotsKinds directory if it exists", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			kotsKindsDir := filepath.Join(dir, "kotsKinds")
			err = os.MkdirAll(kotsKindsDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			path := kotsutil.GetKotsKindsPath(dir)
			Expect(path).To(Equal(kotsKindsDir))
		})

		It("returns the path to the upstream directory if kotsKinds does not exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			upstreamDir := filepath.Join(dir, "upstream")
			err = os.MkdirAll(upstreamDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			path := kotsutil.GetKotsKindsPath(dir)
			Expect(path).To(Equal(upstreamDir))
		})

		It("returns the path to the kotsKinds directory if both kotsKinds and upstream exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			kotsKindsDir := filepath.Join(dir, "kotsKinds")
			err = os.MkdirAll(kotsKindsDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			upstreamDir := filepath.Join(dir, "upstream")
			err = os.MkdirAll(upstreamDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			path := kotsutil.GetKotsKindsPath(dir)
			Expect(path).To(Equal(kotsKindsDir))
		})

		It("returns the path to root directory if neither kotsKinds nor upstream exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			path := kotsutil.GetKotsKindsPath(dir)
			Expect(path).To(Equal(dir))
		})
	})

	Describe("LoadKotsKinds()", func() {
		It("loads kots kinds from 'kotsKinds' directory if exists", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			kotsKindsDir := filepath.Join(dir, "kotsKinds")
			err = os.MkdirAll(kotsKindsDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(kotsKindsDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsKinds, err := kotsutil.LoadKotsKinds(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKinds).ToNot(BeNil())
			Expect(kotsKinds.KotsApplication.Spec.Title).To(Equal("foo"))
		})

		It("loads kots kinds from 'upstream' directory if 'kotsKinds' does not exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			upstreamDir := filepath.Join(dir, "upstream")
			err = os.MkdirAll(upstreamDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(upstreamDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsKinds, err := kotsutil.LoadKotsKinds(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKinds).ToNot(BeNil())
			Expect(kotsKinds.KotsApplication.Spec.Title).To(Equal("foo"))
		})

		It("loads kots kinds from 'kotsKinds' directory if both 'kotsKinds' and 'upstream' exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			kotsKindsDir := filepath.Join(dir, "kotsKinds")
			err = os.MkdirAll(kotsKindsDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			upstreamDir := filepath.Join(dir, "upstream")
			err = os.MkdirAll(upstreamDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(kotsKindsDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(upstreamDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: bar\nspec:\n  title: bar"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsKinds, err := kotsutil.LoadKotsKinds(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKinds).ToNot(BeNil())
			Expect(kotsKinds.KotsApplication.Spec.Title).To(Equal("foo"))
		})

		It("loads kots kinds from root directory if neither 'kotsKinds' nor 'upstream' exist", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = os.WriteFile(filepath.Join(dir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsKinds, err := kotsutil.LoadKotsKinds(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKinds).ToNot(BeNil())
			Expect(kotsKinds.KotsApplication.Spec.Title).To(Equal("foo"))
		})
	})

	Describe("FindKotsAppInPath()", func() {
		It("returns nil if no kots app is found", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = os.WriteFile(filepath.Join(dir, "foo.yaml"), []byte("apiVersion: custom.io/v1beta1\nkind: Foo\nmetadata:\n  name: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsApp, err := kotsutil.FindKotsAppInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsApp).To(BeNil())
		})

		It("returns the kots app if found in the root directory", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = os.WriteFile(filepath.Join(dir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsApp, err := kotsutil.FindKotsAppInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsApp).ToNot(BeNil())
			Expect(kotsApp.Spec.Title).To(Equal("foo"))
		})

		It("returns the kots app if found in a subdirectory", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			subDir := filepath.Join(dir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(subDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsApp, err := kotsutil.FindKotsAppInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsApp).ToNot(BeNil())
			Expect(kotsApp.Spec.Title).To(Equal("foo"))
		})

		It("returns only one kots app if multiple are found", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			subDir := filepath.Join(dir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(dir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: foo\nspec:\n  title: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(subDir, "kots-app.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: bar\nspec:\n  title: bar"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsApp, err := kotsutil.FindKotsAppInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsApp).ToNot(BeNil())
			Expect(kotsApp.Spec.Title).To(Equal("foo"))
		})
	})

	Describe("FindConfigInPath()", func() {
		It("returns nil if no config is found", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = os.WriteFile(filepath.Join(dir, "foo.yaml"), []byte("apiVersion: custom.io/v1beta1\nkind: Foo\nmetadata:\n  name: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsConfig, err := kotsutil.FindConfigInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsConfig).To(BeNil())
		})

		It("returns the config if found in the root directory", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = os.WriteFile(filepath.Join(dir, "kots-config.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Config\nmetadata:\n  name: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsConfig, err := kotsutil.FindConfigInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsConfig).ToNot(BeNil())
			Expect(kotsConfig.ObjectMeta.Name).To(Equal("foo"))
		})

		It("returns the config if found in a subdirectory", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			subDir := filepath.Join(dir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(subDir, "kots-config.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Config\nmetadata:\n  name: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsConfig, err := kotsutil.FindConfigInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsConfig).ToNot(BeNil())
			Expect(kotsConfig.ObjectMeta.Name).To(Equal("foo"))
		})

		It("returns only one config if multiple are found", func() {
			dir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			subDir := filepath.Join(dir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(dir, "kots-config.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Config\nmetadata:\n  name: foo"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(subDir, "kots-config.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Config\nmetadata:\n  name: bar"), 0644)
			Expect(err).ToNot(HaveOccurred())

			kotsConfig, err := kotsutil.FindConfigInPath(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsConfig).ToNot(BeNil())
			Expect(kotsConfig.ObjectMeta.Name).To(Equal("foo"))
		})
	})

	Describe("EncryptConfigValues()", func() {
		It("does not error when the config field is missing", func() {
			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta1.ConfigValues{},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not error when the configValues field is missing", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the configItemType is not found", func() {
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues["name"] = kotsv1beta1.ConfigValue{
				ValuePlaintext: "valuePlaintext",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: "item1",
										Type: "",
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("item type was not found"))
		})

		It("returns an error if the configItemType is not a password", func() {
			configItemType := "notAPassword"
			itemName := "some-item"
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				ValuePlaintext: "valuePlainText",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: itemName,
										Type: configItemType,
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("item type was \"notAPassword\" (not password)"))
		})

		It("encrypts the value if it is a password", func() {
			configItemType := "password"
			itemName := "some-item"
			nonEncryptedValue := "not-encrypted"
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				Value:          nonEncryptedValue,
				ValuePlaintext: "some-nonEncryptedValue-in-plain-text",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: itemName,
										Type: configItemType,
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].Value).ToNot(Equal(nonEncryptedValue))
		})
	})

	Describe("DecryptConfigValues()", func() {
		It("does not error when config values are empty", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not change the value if it is missing", func() {
			itemName := "some-item"
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				Value:          "",
				ValuePlaintext: "some-nonEncryptedValue-in-plain-text",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].Value).To(Equal(""))
		})

		It("decrypts the value if it is present", func() {
			itemName := "some-item"
			encryptedValue := crypto.Encrypt([]byte("someEncryptedValueInPlainText"))
			encodedValue := base64.StdEncoding.EncodeToString(encryptedValue)
			valuePlainText := "someEncryptedValueInPlainText"
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				Value:          encodedValue,
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].Value).To(Equal(""))
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].ValuePlaintext).To(Equal(valuePlainText))
		})

		It("does not change the value if it cannot be decoded", func() {
			itemName := "some-item"
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				Value:          "not-an-encoded-value",
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].Value).To(Equal("not-an-encoded-value"))
		})

		It("does not change the value if it cannot be decrypted", func() {
			itemName := "some-item"
			encodedButNotEncryptedValue := base64.StdEncoding.EncodeToString([]byte("someEncryptedValueInPlainText"))
			configValues := make(map[string]kotsv1beta1.ConfigValue)
			configValues[itemName] = kotsv1beta1.ConfigValue{
				Value:          encodedButNotEncryptedValue,
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta1.ConfigValues{
					Spec: kotsv1beta1.ConfigValuesSpec{
						Values: configValues,
					},
				},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(kotsKind.ConfigValues.Spec.Values[itemName].Value).To(Equal(encodedButNotEncryptedValue))
		})
	})

	Describe("IsConfigurable()", func() {
		It("returns false when the client-side object is not set", func() {
			var kotsKind *kotsutil.KotsKinds = nil
			preflightResult := kotsKind.IsConfigurable()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns false when the client-side object does not have config set", func() {
			kotsKind := &kotsutil.KotsKinds{}
			preflightResult := kotsKind.IsConfigurable()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns false when the length of groups is zero", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{},
					},
				},
			}
			preflightResult := kotsKind.IsConfigurable()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns true when the length of the groups is greater than zero", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group-item",
							},
						},
					},
				},
			}
			preflightResult := kotsKind.IsConfigurable()
			Expect(preflightResult).To(BeTrue())
		})
	})

	Describe("HasPreflights()", func() {
		It("returns false when the client-side object is not set", func() {
			var kotsKind *kotsutil.KotsKinds = nil
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns false when the client-side object does not have preflights set", func() {
			kotsKind := &kotsutil.KotsKinds{}
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns false when the client-side object does not have analyzers", func() {
			kotsKind := &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{},
					},
				},
			}
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns true when there are analyzers defined in the preflight spec", func() {
			// single spec
			kotsKind := &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{},
						},
					},
				},
			}
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeTrue())
		})

		It("returns false when all analyzers are excluded", func() {
			kotsKind := &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(true),
									},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(true),
									},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(true),
									},
								},
							},
						},
					},
				},
			}
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns true when a single analyzer is not excluded", func() {
			kotsKind := &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(true),
									},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(false),
									},
								},
							},
						},
					},
				},
			}
			preflightResult := kotsKind.HasPreflights()
			Expect(preflightResult).To(BeTrue())
		})
	})

	Describe("LoadBrandingArchiveFromPath()", func() {
		It("returns an error when the path does not exist", func() {
			_, err := kotsutil.LoadBrandingArchiveFromPath("/does/not/exist")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when the path is not a directory", func() {
			tmpFile, err := ioutil.TempFile("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = kotsutil.LoadBrandingArchiveFromPath(tmpFile.Name())
			Expect(err).To(HaveOccurred())
		})

		It("returns an empty branding archive when there are no branding files", func() {
			tmpDir, err := ioutil.TempDir("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			archive, err := kotsutil.LoadBrandingArchiveFromPath(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(archive.Len()).To(Equal(0))
		})

		It("returns a branding archive when the path contains branding files", func() {
			tmpDir, err := ioutil.TempDir("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			err = ioutil.WriteFile(filepath.Join(tmpDir, "branding.css"), []byte("body { background-color: red; }"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(tmpDir, "font.ttf"), []byte("my-font-data"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(tmpDir, "application.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: app-slug\nspec:\n  icon: https://foo.com/icon.png\n  title: App Name"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(tmpDir, "random.yaml"), []byte("some: yaml"), 0644)
			Expect(err).ToNot(HaveOccurred())

			archive, err := kotsutil.LoadBrandingArchiveFromPath(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(archive).ToNot(BeNil())

			b, err := util.GetFileFromTGZArchive(archive, "branding.css")
			Expect(err).ToNot(HaveOccurred())
			Expect(b.String()).To(Equal("body { background-color: red; }"))

			archive, err = kotsutil.LoadBrandingArchiveFromPath(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(archive).ToNot(BeNil())

			b, err = util.GetFileFromTGZArchive(archive, "font.ttf")
			Expect(err).ToNot(HaveOccurred())
			Expect(b.String()).To(Equal("my-font-data"))

			archive, err = kotsutil.LoadBrandingArchiveFromPath(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(archive).ToNot(BeNil())

			b, err = util.GetFileFromTGZArchive(archive, "application.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(b.String()).To(Equal("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: app-slug\nspec:\n  icon: https://foo.com/icon.png\n  title: App Name"))

			_, err = util.GetFileFromTGZArchive(archive, "random.yaml")
			Expect(err).To(HaveOccurred())
		})
	})
})

func TestIsKotsKind(t *testing.T) {
	type args struct {
		apiVersion string
		kind       string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "velero backup",
			args: args{
				apiVersion: "velero.io/v1",
				kind:       "Backup",
			},
			want: true,
		},
		{
			name: "velero other",
			args: args{
				apiVersion: "velero.io/v1",
				kind:       "Blah",
			},
			want: false,
		},
		{
			name: "kots.io/v1beta1",
			args: args{
				apiVersion: "kots.io/v1beta1",
			},
			want: true,
		},
		{
			name: "kots.io/v1beta2",
			args: args{
				apiVersion: "kots.io/v1beta2",
			},
			want: true,
		},
		{
			name: "troubleshoot.sh/v1beta2",
			args: args{
				apiVersion: "troubleshoot.sh/v1beta2",
			},
			want: true,
		},
		{
			name: "troubleshoot.replicated.com/v1beta1",
			args: args{
				apiVersion: "troubleshoot.replicated.com/v1beta1",
			},
			want: true,
		},
		{
			name: "cluster.kurl.sh/v1beta1",
			args: args{
				apiVersion: "cluster.kurl.sh/v1beta1",
			},
			want: true,
		},
		{
			name: "kurl.sh/v1beta1",
			args: args{
				apiVersion: "kurl.sh/v1beta1",
			},
			want: true,
		},
		{
			name: "app.k8s.io/v1beta1",
			args: args{
				apiVersion: "app.k8s.io/v1beta1",
			},
			want: true,
		},
		{
			name: "app.unknown.io/v1",
			args: args{
				apiVersion: "app.unknown.io/v1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := kotsutil.IsKotsKind(tt.args.apiVersion, tt.args.kind); got != tt.want {
				t.Errorf("IsKotsKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetImagesFromKotsKinds(t *testing.T) {
	type args struct {
		kotsKinds    *kotsutil.KotsKinds
		destRegistry *dockerregistrytypes.RegistryOptions
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				kotsKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AdditionalImages: []string{
								"registry.replicated.com/appslug/image:version",
							},
						},
					},
					Preflight: &troubleshootv1beta2.Preflight{
						Spec: troubleshootv1beta2.PreflightSpec{
							Collectors: []*troubleshootv1beta2.Collect{
								{
									Run: &troubleshootv1beta2.Run{
										Image: "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
									},
								},
								{
									RunPod: &troubleshootv1beta2.RunPod{
										PodSpec: corev1.PodSpec{
											Containers: []corev1.Container{
												{
													Image: "nginx:1",
												},
											},
										},
									},
								},
							},
						},
					},
					SupportBundle: &troubleshootv1beta2.SupportBundle{
						Spec: troubleshootv1beta2.SupportBundleSpec{
							Collectors: []*troubleshootv1beta2.Collect{
								{
									Run: &troubleshootv1beta2.Run{
										Image: "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
									},
								},
							},
						},
					},
				},
			},
			want: []string{
				"registry.replicated.com/appslug/image:version",
				"quay.io/replicatedcom/qa-kots-1:alpine-3.5",
				"nginx:1",
				"quay.io/replicatedcom/qa-kots-2:alpine-3.4",
			},
		},
		{
			name: "excludes images that already point to the destination registry",
			args: args{
				kotsKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AdditionalImages: []string{
								"registry.replicated.com/appslug/image:version",
							},
						},
					},
					Preflight: &troubleshootv1beta2.Preflight{
						Spec: troubleshootv1beta2.PreflightSpec{
							Collectors: []*troubleshootv1beta2.Collect{
								{
									Run: &troubleshootv1beta2.Run{
										Image: "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
									},
								},
								{
									Run: &troubleshootv1beta2.Run{
										Image: "testing.registry.com:5000/testing-ns/random-image:2",
									},
								},
								{
									RunPod: &troubleshootv1beta2.RunPod{
										PodSpec: corev1.PodSpec{
											Containers: []corev1.Container{
												{
													Image: "nginx:1",
												},
											},
										},
									},
								},
							},
						},
					},
					SupportBundle: &troubleshootv1beta2.SupportBundle{
						Spec: troubleshootv1beta2.SupportBundleSpec{
							Collectors: []*troubleshootv1beta2.Collect{
								{
									Run: &troubleshootv1beta2.Run{
										Image: "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
									},
								},
								{
									Run: &troubleshootv1beta2.Run{
										Image: "testing.registry.com:5000/testing-ns/random-image:1",
									},
								},
							},
						},
					},
				},
				destRegistry: &dockerregistrytypes.RegistryOptions{
					Endpoint:  "testing.registry.com:5000",
					Namespace: "testing-ns",
					Username:  "testing-user-name",
					Password:  "testing-password",
				},
			},
			want: []string{
				"registry.replicated.com/appslug/image:version",
				"quay.io/replicatedcom/qa-kots-1:alpine-3.5",
				"nginx:1",
				"quay.io/replicatedcom/qa-kots-2:alpine-3.4",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, err := kotsutil.GetImagesFromKotsKinds(tt.args.kotsKinds, tt.args.destRegistry)
			req.NoError(err)

			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestKotsKinds_Marshal(t *testing.T) {
	type fields struct {
		KotsApplication       kotsv1beta1.Application
		Application           *applicationv1beta1.Application
		V1Beta1HelmCharts     *kotsv1beta1.HelmChartList
		V1Beta2HelmCharts     *kotsv1beta2.HelmChartList
		Collector             *troubleshootv1beta2.Collector
		Preflight             *troubleshootv1beta2.Preflight
		Analyzer              *troubleshootv1beta2.Analyzer
		SupportBundle         *troubleshootv1beta2.SupportBundle
		Redactor              *troubleshootv1beta2.Redactor
		HostPreflight         *troubleshootv1beta2.HostPreflight
		Config                *kotsv1beta1.Config
		ConfigValues          *kotsv1beta1.ConfigValues
		Installation          kotsv1beta1.Installation
		License               *kotsv1beta1.License
		Identity              *kotsv1beta1.Identity
		IdentityConfig        *kotsv1beta1.IdentityConfig
		Backup                *velerov1.Backup
		Installer             *kurlv1beta1.Installer
		LintConfig            *kotsv1beta1.LintConfig
		EmbeddedClusterConfig *embeddedclusterv1beta1.Config
	}
	type args struct {
		g string
		v string
		k string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		preInit func(t *testing.T)
		want    string
	}{
		{
			name: "backup exists, not EC",
			fields: fields{
				Backup: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-name"},
					TypeMeta:   metav1.TypeMeta{APIVersion: "velero.io/v1", Kind: "Backup"},
				},
			},
			args: args{
				g: "velero.io",
				v: "v1",
				k: "Backup",
			},
			want: `apiVersion: velero.io/v1
kind: Backup
metadata:
  creationTimestamp: null
  name: backup-name
spec:
  csiSnapshotTimeout: 0s
  hooks: {}
  itemOperationTimeout: 0s
  metadata: {}
  ttl: 0s
status: {}
`,
		},
		{
			name: "no backup exists, not EC",
			args: args{
				g: "velero.io",
				v: "v1",
				k: "Backup",
			},
			want: "",
		},
		{
			name: "no backup exists, EC",
			preInit: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "test")
			},
			args: args{
				g: "velero.io",
				v: "v1",
				k: "Backup",
			},
			want: "",
		},
		{
			name: "backup exists, EC",
			fields: fields{
				Backup: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-name"},
					TypeMeta:   metav1.TypeMeta{APIVersion: "velero.io/v1", Kind: "Backup"},
				},
			},
			preInit: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "test")
			},
			args: args{
				g: "velero.io",
				v: "v1",
				k: "Backup",
			},
			want: `apiVersion: velero.io/v1
kind: Backup
metadata:
  creationTimestamp: null
  name: backup-name
spec:
  csiSnapshotTimeout: 0s
  hooks: {}
  itemOperationTimeout: 0s
  metadata: {}
  ttl: 0s
status: {}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := kotsutil.KotsKinds{
				KotsApplication:       tt.fields.KotsApplication,
				Application:           tt.fields.Application,
				V1Beta1HelmCharts:     tt.fields.V1Beta1HelmCharts,
				V1Beta2HelmCharts:     tt.fields.V1Beta2HelmCharts,
				Collector:             tt.fields.Collector,
				Preflight:             tt.fields.Preflight,
				Analyzer:              tt.fields.Analyzer,
				SupportBundle:         tt.fields.SupportBundle,
				Redactor:              tt.fields.Redactor,
				HostPreflight:         tt.fields.HostPreflight,
				Config:                tt.fields.Config,
				ConfigValues:          tt.fields.ConfigValues,
				Installation:          tt.fields.Installation,
				License:               tt.fields.License,
				Identity:              tt.fields.Identity,
				IdentityConfig:        tt.fields.IdentityConfig,
				Backup:                tt.fields.Backup,
				Installer:             tt.fields.Installer,
				LintConfig:            tt.fields.LintConfig,
				EmbeddedClusterConfig: tt.fields.EmbeddedClusterConfig,
			}

			req := require.New(t)
			if tt.preInit != nil {
				tt.preInit(t)
			}
			got, err := o.Marshal(tt.args.g, tt.args.v, tt.args.k)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func TestFindChannelIDInLicense(t *testing.T) {
	tests := []struct {
		name              string
		license           *kotsv1beta1.License
		requestedSlug     string
		expectedChannelID string
		expectError       bool
	}{
		{
			name: "Found slug",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:   "channel-id-1",
							ChannelSlug: "slug-1",
							IsDefault:   true,
						},
						{
							ChannelID:   "channel-id-2",
							ChannelSlug: "slug-2",
							IsDefault:   false,
						},
					},
				},
			},
			requestedSlug:     "slug-2",
			expectedChannelID: "channel-id-2",
			expectError:       false,
		},
		{
			name: "Empty requested slug",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID: "top-level-channel-id",
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:   "channel-id-1",
							ChannelSlug: "channel-slug-1",
						},
						{
							ChannelID:   "channel-id-2",
							ChannelSlug: "channel-slug-2",
						},
					},
				},
			},
			requestedSlug:     "",
			expectedChannelID: "top-level-channel-id",
			expectError:       false,
		},
		{
			name: "Legacy license with no / empty channels",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID: "test-channel-id",
				},
			},
			requestedSlug:     "test-slug",
			expectedChannelID: "test-channel-id",
			expectError:       false,
		},
		{
			name: "No matching slug should error",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID: "top-level-channel-id",
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:   "channel-id-1",
							ChannelSlug: "channel-slug-1",
						},
						{
							ChannelID:   "channel-id-2",
							ChannelSlug: "channel-slug-2",
						},
					},
				},
			},
			requestedSlug:     "non-existent-slug",
			expectedChannelID: "",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelID, err := kotsutil.FindChannelIDInLicense(tt.requestedSlug, tt.license)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedChannelID, channelID)
			}
		})
	}
}

func TestFindChannelInLicense(t *testing.T) {
	tests := []struct {
		name            string
		license         *kotsv1beta1.License
		requestedID     string
		expectedChannel *kotsv1beta1.Channel
		expectError     bool
	}{
		{
			name: "Find multi channel license",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:        "channel-id-1",
							ChannelName:      "name-1",
							IsDefault:        true,
							IsSemverRequired: true,
						},
						{
							ChannelID:        "channel-id-2",
							ChannelName:      "name-2",
							IsDefault:        false,
							IsSemverRequired: false,
						},
					},
				},
			},
			requestedID: "channel-id-2",
			expectedChannel: &kotsv1beta1.Channel{
				ChannelID:        "channel-id-2",
				ChannelName:      "name-2",
				IsDefault:        false,
				IsSemverRequired: false,
			},
			expectError: false,
		},
		{
			name: "Legacy license with no / empty channels",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID:        "test-channel-id",
					ChannelName:      "test-channel-name",
					IsSemverRequired: true,
				},
			},
			requestedID: "test-channel-id",
			expectedChannel: &kotsv1beta1.Channel{
				ChannelID:        "test-channel-id",
				ChannelName:      "test-channel-name",
				IsSemverRequired: true,
				IsDefault:        true,
			},
			expectError: false,
		},
		{
			name: "No matching ID should error",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID:        "channel-id-1",
					ChannelName:      "name-1",
					IsSemverRequired: true,
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:        "channel-id-1",
							ChannelName:      "name-1",
							IsDefault:        true,
							IsSemverRequired: true,
						},
						{
							ChannelID:        "channel-id-2",
							ChannelName:      "name-2",
							IsDefault:        false,
							IsSemverRequired: false,
						},
					},
				},
			},
			requestedID:     "non-existent-id",
			expectedChannel: nil,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := kotsutil.FindChannelInLicense(tt.requestedID, tt.license)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, channel)
				require.Equal(t, tt.expectedChannel, channel)
			}
		})
	}
}

func TestGetInstallationParamsWithClientset(t *testing.T) {
	type args struct {
		configMapName string
		namespace     string
		clientSet     kubernetes.Interface
	}
	tests := []struct {
		name    string
		args    args
		want    kotsutil.InstallationParams
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "basic test",
			args: args{
				configMapName: "kotsadm",
				namespace:     "test-namespace",
				clientSet: fake.NewClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						"additional-annotations":    "abc/xyz=test-annotation1,test.annotation/two=test.value/two/test",
						"additional-labels":         "xyz=label2,abc=123",
						"app-version-label":         "",
						"ensure-rbac":               "true",
						"initial-app-images-pushed": "false",
						"kots-install-id":           "2liAJUuyAi3Gnyhvi5Arv5BJRZ4",
						"minio-enabled-snapshots":   "true",
						"registry-is-read-only":     "false",
						"requested-channel-slug":    "stable",
						"skip-compatibility-check":  "false",
						"skip-preflights":           "false",
						"skip-rbac-check":           "false",
						"strict-security-context":   "false",
						"use-minimal-rbac":          "false",
						"wait-duration":             "2m0s",
						"with-minio":                "true",
					},
				}),
			},
			want: kotsutil.InstallationParams{
				AdditionalAnnotations:  map[string]string{"abc/xyz": "test-annotation1", "test.annotation/two": "test.value/two/test"},
				AdditionalLabels:       map[string]string{"abc": "123", "xyz": "label2"},
				AppVersionLabel:        "",
				EnsureRBAC:             true,
				KotsadmRegistry:        "",
				SkipImagePush:          false,
				SkipPreflights:         false,
				SkipCompatibilityCheck: false,
				RegistryIsReadOnly:     false,
				EnableImageDeletion:    false,
				SkipRBACCheck:          false,
				UseMinimalRBAC:         false,
				StrictSecurityContext:  false,
				WaitDuration:           time.Minute * 2,
				WithMinio:              true,
				RequestedChannelSlug:   "stable",
			},
		},
		{
			name: "no labels or annotations",
			args: args{
				configMapName: "kotsadm",
				namespace:     "test-namespace",
				clientSet: fake.NewClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						"app-version-label":         "",
						"ensure-rbac":               "true",
						"initial-app-images-pushed": "false",
						"kots-install-id":           "2liAJUuyAi3Gnyhvi5Arv5BJRZ4",
						"minio-enabled-snapshots":   "true",
						"registry-is-read-only":     "false",
						"requested-channel-slug":    "stable",
						"skip-compatibility-check":  "false",
						"skip-preflights":           "false",
						"skip-rbac-check":           "false",
						"strict-security-context":   "false",
						"use-minimal-rbac":          "false",
						"wait-duration":             "2m0s",
						"with-minio":                "true",
					},
				}),
			},
			want: kotsutil.InstallationParams{
				AdditionalAnnotations:  map[string]string{},
				AdditionalLabels:       map[string]string{},
				AppVersionLabel:        "",
				EnsureRBAC:             true,
				KotsadmRegistry:        "",
				SkipImagePush:          false,
				SkipPreflights:         false,
				SkipCompatibilityCheck: false,
				RegistryIsReadOnly:     false,
				EnableImageDeletion:    false,
				SkipRBACCheck:          false,
				UseMinimalRBAC:         false,
				StrictSecurityContext:  false,
				WaitDuration:           time.Minute * 2,
				WithMinio:              true,
				RequestedChannelSlug:   "stable",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := kotsutil.GetInstallationParamsWithClientset(tt.args.clientSet, tt.args.configMapName, tt.args.namespace)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
