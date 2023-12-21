package kotsutil_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
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
				ConfigValues: &kotsv1beta.ConfigValues{},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not error when the configValues field is missing", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta.Config{},
			}
			err := kotsKind.EncryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the configItemType is not found", func() {
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues["name"] = kotsv1beta.ConfigValue{
				ValuePlaintext: "valuePlaintext",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta.Config{
					Spec: kotsv1beta.ConfigSpec{
						Groups: []kotsv1beta.ConfigGroup{
							{
								Items: []kotsv1beta.ConfigItem{
									{
										Name: "item1",
										Type: "",
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				ValuePlaintext: "valuePlainText",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta.Config{
					Spec: kotsv1beta.ConfigSpec{
						Groups: []kotsv1beta.ConfigGroup{
							{
								Items: []kotsv1beta.ConfigItem{
									{
										Name: itemName,
										Type: configItemType,
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				Value:          nonEncryptedValue,
				ValuePlaintext: "some-nonEncryptedValue-in-plain-text",
			}

			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta.Config{
					Spec: kotsv1beta.ConfigSpec{
						Groups: []kotsv1beta.ConfigGroup{
							{
								Items: []kotsv1beta.ConfigItem{
									{
										Name: itemName,
										Type: configItemType,
									},
								},
							},
						},
					},
				},
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
				Config: &kotsv1beta.Config{},
			}
			err := kotsKind.DecryptConfigValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not change the value if it is missing", func() {
			itemName := "some-item"
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				Value:          "",
				ValuePlaintext: "some-nonEncryptedValue-in-plain-text",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				Value:          encodedValue,
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				Value:          "not-an-encoded-value",
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
			configValues := make(map[string]kotsv1beta.ConfigValue)
			configValues[itemName] = kotsv1beta.ConfigValue{
				Value:          encodedButNotEncryptedValue,
				ValuePlaintext: "",
			}

			kotsKind := &kotsutil.KotsKinds{
				ConfigValues: &kotsv1beta.ConfigValues{
					Spec: kotsv1beta.ConfigValuesSpec{
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
				Config: &kotsv1beta.Config{
					Spec: kotsv1beta.ConfigSpec{
						Groups: []kotsv1beta.ConfigGroup{},
					},
				},
			}
			preflightResult := kotsKind.IsConfigurable()
			Expect(preflightResult).To(BeFalse())
		})

		It("returns true when the length of the groups is greater than zero", func() {
			kotsKind := &kotsutil.KotsKinds{
				Config: &kotsv1beta.Config{
					Spec: kotsv1beta.ConfigSpec{
						Groups: []kotsv1beta.ConfigGroup{
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

		It("returns true when there are more than one analyzers defined in the preflight spec", func() {
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
	})

	Describe("GetKustomizeBinaryPath()", func() {
		It("returns unusable path 'kustomize' if the Kustomize Version cannot be found", func() {
			kotsKind := &kotsutil.KotsKinds{}

			binaryPath := kotsKind.GetKustomizeBinaryPath()
			Expect(binaryPath).To(Equal("kustomize"))
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
