package kotsutil_test

import (
	"encoding/base64"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

var _ = Describe("Kots", func() {
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
			tmpFile, err := os.CreateTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = kotsutil.LoadBrandingArchiveFromPath(tmpFile.Name())
			Expect(err).To(HaveOccurred())
		})

		It("returns an empty branding archive when there are no branding files", func() {
			tmpDir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			archive, err := kotsutil.LoadBrandingArchiveFromPath(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(archive.Len()).To(Equal(0))
		})

		It("returns a branding archive when the path contains branding files", func() {
			tmpDir, err := os.MkdirTemp("", "kotsutil-test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			err = os.WriteFile(filepath.Join(tmpDir, "branding.css"), []byte("body { background-color: red; }"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(tmpDir, "font.ttf"), []byte("my-font-data"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(tmpDir, "application.yaml"), []byte("apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: app-slug\nspec:\n  icon: https://foo.com/icon.png\n  title: App Name"), 0644)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(tmpDir, "random.yaml"), []byte("some: yaml"), 0644)
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

	Describe("GenUniqueKotsKindsFilename()", func() {
		It("returns the same name when there are no file entries", func() {
			filename := kotsutil.GenUniqueKotsKindFilename(nil, "unique")
			Expect(filename).To(Equal("unique.yaml"))

			tmpRendered := map[string][]byte{}
			filename = kotsutil.GenUniqueKotsKindFilename(tmpRendered, "unique")
			Expect(filename).To(Equal("unique.yaml"))
		})

		It("returns the same name when there is no conflict", func() {
			tmpRendered := map[string][]byte{
				"random.yaml":  nil,
				"example.yaml": nil,
			}

			filename := kotsutil.GenUniqueKotsKindFilename(tmpRendered, "unique")
			Expect(filename).To(Equal("unique.yaml"))
		})

		It("returns a unique name when there is a conflict", func() {
			tmpRendered := map[string][]byte{
				"unique.yaml":  nil,
				"example.yaml": nil,
			}

			filename := kotsutil.GenUniqueKotsKindFilename(tmpRendered, "unique")
			Expect(filename).To(Equal("unique-1.yaml"))
		})

		It("returns a unique name when there is a conflict and the generated name creates a new conflict", func() {
			tmpRendered := map[string][]byte{
				"unique.yaml":   nil,
				"unique-1.yaml": nil,
				"unique-2.yaml": nil,
				"unique-4.yaml": nil,
				"example.yaml":  nil,
			}

			filename := kotsutil.GenUniqueKotsKindFilename(tmpRendered, "unique")
			Expect(filename).To(Equal("unique-3.yaml"))
		})
	})
})
