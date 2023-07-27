package secrets_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/kots/pkg/secrets"
	"github.com/replicatedhq/kots/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Secrets", func() {
	Describe("ReplaceSecretsInPath", func() {
		var (
			tmpFile       *os.File
			tmpArchiveDir string
			clientset     *fake.Clientset
			namespace     string
			validLabels   map[string]string
		)

		BeforeEach(func() {
			util.PodNamespace = "test-namespace"
			namespace = util.PodNamespace

			validLabels = make(map[string]string)
			validLabels["kots.io/buildphase"] = "secret"
			validLabels["kots.io/secrettype"] = "sealedsecrets"

			var err error
			tmpArchiveDir, err = os.MkdirTemp(".", "secrets-test")
			Expect(err).ToNot(HaveOccurred())

			tmpFile, err = os.CreateTemp(tmpArchiveDir, "secrets-test")
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			err := os.RemoveAll(tmpArchiveDir)
			Expect(err).ToNot(HaveOccurred())

			err = os.RemoveAll(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not return an error if there are no secrets found", func() {
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items:    nil,
			})
			err := secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if there are multiple secret buildphases", func() {
			var labels = make(map[string]string)
			labels["kots.io/buildphase"] = "secret"
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    labels,
					},
				}, {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret2",
						Namespace: namespace,
						Labels:    labels,
					},
				}},
			})

			err := secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple secret buildphases are not supported"))
		})

		It("returns an error if the secret type is not supported", func() {
			var labels = make(map[string]string)
			labels["kots.io/buildphase"] = "secret"
			labels["kots.io/secrettype"] = "unsupported-type"
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    labels,
					},
				}},
			})

			err := secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown secret type"))
		})

		It("returns an error if the archivePath could not be Walk'd through", func() {
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
				}},
			})

			err := secrets.ReplaceSecretsInPath("invalid-archive-dir", clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get secrets in path"))
			Expect(err.Error()).To(ContainSubstring("could not walk through the archive directory"))
		})

		It("does not return an error secretPaths is empty", func() {
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
				}},
			})

			err := secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not error if it cannot decode the contents of the 'secret' file", func() {
			var data = make(map[string][]byte)
			data["cert.pem"] = []byte(validPublicKey)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
					Data: data,
				}},
			})

			invalidSecret := `invalid yaml`
			_, err := tmpFile.WriteString(invalidSecret)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())

			secretContents, err := os.ReadFile(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(string(secretContents)).To(ContainSubstring("invalid yaml"))
		})

		It("does not update the secret if the apiVersion is not v1", func() {
			var data = make(map[string][]byte)
			data["cert.pem"] = []byte(validPublicKey)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
					Data: data,
				}},
			})

			err, secret := writeSecret("extensions/v1beta1", "PodSecurityPolicy", namespace, false, false, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())

			secretContents, err := os.ReadFile(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(string(secretContents)).To(Equal(secret))
		})

		It("does not update the secret if the kind is not secret", func() {
			var data = make(map[string][]byte)
			data["cert.pem"] = []byte(validPublicKey)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
					Data: data,
				}},
			})

			err, secret := writeSecret("v1", "Pod", namespace, false, false, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())

			secretContents, err := os.ReadFile(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(string(secretContents)).To(Equal(secret))
		})

		It("does not update the secret if the labels are not correct", func() {
			var labels = make(map[string]string)
			labels["notASecretLabel"] = "also-not-a-secret"
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    labels,
					},
				}},
			})

			wronglyLabeledSecret := fmt.Sprintf(`---
apiVersion: v1
kind: "Secret"
metadata:
  name: secret1
  namespace: %s
  labels:
    notASecretLabel: also-not-a-secret`, namespace)
			_, err := tmpFile.WriteString(wronglyLabeledSecret)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())

			secretContents, err := os.ReadFile(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(string(secretContents)).To(ContainSubstring(wronglyLabeledSecret))
		})

		It("returns an error if the certificate key is missing", func() {
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
				}},
			})

			err, _ := writeSecret("v1", "Secret", namespace, false, false, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to read public key from secret"))
		})

		It("returns an error if the certificate key cannot be parsed from the clientset", func() {
			var data = make(map[string][]byte)
			invalidPub := generateInvalidPubKey()
			data["cert.pem"] = []byte(invalidPub)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
					Data: data,
				}},
			})

			err, _ := writeSecret("v1", "Secret", namespace, false, false, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse certificate"))
		})

		It("returns an error if a sealedsecret cannot be created", func() {
			util.PodNamespace = "" // if there is no namespace set, an error will be returned

			var data = make(map[string][]byte)
			data["cert.pem"] = []byte(validPublicKey)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "secret1",
						Labels: validLabels,
					},
					Data: data,
				}},
			})

			err, _ := writeSecret("v1", "Secret", "", false, true, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create sealedsecret"))
		})

		Context("secret namespace is not provided", func() {
			Context("env var DEV_NAMESPACE is set", func() {
				var devNamespace = "dev-namespace"

				BeforeEach(func() {
					util.PodNamespace = ""
					err := os.Setenv("DEV_NAMESPACE", devNamespace)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Unsetenv("DEV_NAMESPACE")
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not error", func() {
					var data = make(map[string][]byte)
					data["cert.pem"] = []byte(validPublicKey)
					clientset = fake.NewSimpleClientset(&v1.SecretList{
						TypeMeta: metav1.TypeMeta{},
						ListMeta: metav1.ListMeta{},
						Items: []v1.Secret{{
							TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "secret1",
								Namespace: devNamespace,
								Labels:    validLabels,
							},
							Data: data,
						}},
					})

					err, _ := writeSecret("v1", "Secret", "", false, false, tmpFile, 1)
					Expect(err).ToNot(HaveOccurred())

					err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
					Expect(err).ToNot(HaveOccurred())

					secretContents, err := os.ReadFile(tmpFile.Name())
					Expect(err).ToNot(HaveOccurred())
					Expect(string(secretContents)).To(ContainSubstring("kind: SealedSecret"))
					Expect(string(secretContents)).To(ContainSubstring(fmt.Sprintf("namespace: %s", devNamespace)))
				})
			})

			Context("PodNamespace is set", func() {
				var podNamespace = "pod-namespace"

				BeforeEach(func() {
					util.PodNamespace = podNamespace
				})

				AfterEach(func() {
					util.PodNamespace = ""
				})

				It("does not error", func() {

					var data = make(map[string][]byte)
					data["cert.pem"] = []byte(validPublicKey)
					clientset = fake.NewSimpleClientset(&v1.SecretList{
						TypeMeta: metav1.TypeMeta{},
						ListMeta: metav1.ListMeta{},
						Items: []v1.Secret{{
							TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "secret1",
								Labels:    validLabels,
								Namespace: podNamespace,
							},
							Data: data,
						}},
					})

					err, _ := writeSecret("v1", "Secret", "", false, false, tmpFile, 1)
					Expect(err).ToNot(HaveOccurred())

					err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
					Expect(err).ToNot(HaveOccurred())

					secretContents, err := os.ReadFile(tmpFile.Name())
					Expect(err).ToNot(HaveOccurred())
					Expect(string(secretContents)).To(ContainSubstring("kind: SealedSecret"))
					Expect(string(secretContents)).To(ContainSubstring(fmt.Sprintf("namespace: %s", podNamespace)))
				})
			})
		})

		It("updates the secret to a SealedSecret when the secret is valid", func() {
			var data = make(map[string][]byte)
			data["cert.pem"] = []byte(validPublicKey)
			clientset = fake.NewSimpleClientset(&v1.SecretList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []v1.Secret{{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: namespace,
						Labels:    validLabels,
					},
					Data: data,
				}},
			})

			err, _ := writeSecret("v1", "Secret", namespace, false, false, tmpFile, 1)
			Expect(err).ToNot(HaveOccurred())

			err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
			Expect(err).ToNot(HaveOccurred())

			secretContents, err := os.ReadFile(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(string(secretContents)).To(ContainSubstring("kind: SealedSecret"))
			Expect(string(secretContents)).To(ContainSubstring("namespace: test-namespace"))
		})

		Context("multiple secrets files", func() {
			var (
				secretsFiles []*os.File
			)

			BeforeEach(func() {
				for i := 0; i < 10; i++ {
					filePattern := fmt.Sprintf("secrets-test-%d", i)
					tmpFile, err := os.CreateTemp(tmpArchiveDir, filePattern)
					Expect(err).ToNot(HaveOccurred())

					secretsFiles = append(secretsFiles, tmpFile)
				}
			})

			AfterEach(func() {
				for _, file := range secretsFiles {
					err := os.RemoveAll(file.Name())
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("updates multiple secrets to SealedSecrets if they exist", func() {
				var data = make(map[string][]byte)
				data["cert.pem"] = []byte(validPublicKey)
				clientset = fake.NewSimpleClientset(&v1.SecretList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Secret{{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret1",
							Namespace: namespace,
							Labels:    validLabels,
						},
						Data: data,
					}},
				})

				for i := 0; i < len(secretsFiles); i++ {
					err, _ := writeSecret("v1", "Secret", namespace, false, false, secretsFiles[i], 1)
					Expect(err).ToNot(HaveOccurred())
				}

				err := secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
				Expect(err).ToNot(HaveOccurred())

				for i := 0; i < len(secretsFiles); i++ {
					secretContents, err := os.ReadFile(secretsFiles[0].Name())
					Expect(err).ToNot(HaveOccurred())
					Expect(string(secretContents)).To(ContainSubstring("kind: SealedSecret"))
					Expect(string(secretContents)).To(ContainSubstring("namespace: test-namespace"))
				}
			})
		})

		Context("multidoc yaml", func() {
			It("updates both secrets to SealedSecret when the secret is valid", func() {
				var data = make(map[string][]byte)
				data["cert.pem"] = []byte(validPublicKey)
				clientset = fake.NewSimpleClientset(&v1.SecretList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Secret{{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret1",
							Namespace: namespace,
							Labels:    validLabels,
						},
						Data: data,
					}},
				})

				err, _ := writeSecret("v1", "Secret", namespace, false, false, tmpFile, 2)
				Expect(err).ToNot(HaveOccurred())

				transformedSecret1 := `---
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test-secret-0
  namespace: test-namespace`
				transformedSecret2 := `---
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace`

				err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
				Expect(err).ToNot(HaveOccurred())

				secretContents, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())
				Expect(string(secretContents)).To(ContainSubstring(transformedSecret1))
				Expect(string(secretContents)).To(ContainSubstring(transformedSecret2))
			})
			It("updates only SealedSecrets if multiple types are present", func() {
				var data = make(map[string][]byte)
				data["cert.pem"] = []byte(validPublicKey)
				clientset = fake.NewSimpleClientset(&v1.SecretList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Secret{{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret1",
							Namespace: namespace,
							Labels:    validLabels,
						},
						Data: data,
					}},
				})

				notSecretContents := `---
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: test-pod
  namespace: test-namespace
`
				originalSecretContents := `---
apiVersion: v1
kind: Secret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace
  labels:
    kots.io/buildphase: secret
    kots.io/secrettype: sealedsecrets`

				_, err := tmpFile.WriteString(notSecretContents)
				Expect(err).ToNot(HaveOccurred())
				_, err = tmpFile.WriteString(originalSecretContents)
				Expect(err).ToNot(HaveOccurred())

				err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
				Expect(err).ToNot(HaveOccurred())

				transformedSecret := `---
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace`
				updatedFile, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())
				Expect(string(updatedFile)).To(ContainSubstring(transformedSecret))
				Expect(string(updatedFile)).ToNot(ContainSubstring(originalSecretContents))
				Expect(string(updatedFile)).To(ContainSubstring(notSecretContents))
			})
			It("updates only SealedSecrets and leaves types that cannot be decoded", func() {
				var data = make(map[string][]byte)
				data["cert.pem"] = []byte(validPublicKey)
				clientset = fake.NewSimpleClientset(&v1.SecretList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Secret{{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret1",
							Namespace: namespace,
							Labels:    validLabels,
						},
						Data: data,
					}},
				})

				wrongApiVersionForType := `---
apiVersion: bitnami.com/v1alpha1
kind: Pod
metadata:
  creationTimestamp: null
  name: test-pod
  namespace: test-namespace
`
				originalSecretContents := `---
apiVersion: v1
kind: Secret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace
  labels:
    kots.io/buildphase: secret
    kots.io/secrettype: sealedsecrets`

				_, err := tmpFile.WriteString(wrongApiVersionForType)
				Expect(err).ToNot(HaveOccurred())
				_, err = tmpFile.WriteString(originalSecretContents)
				Expect(err).ToNot(HaveOccurred())

				err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
				Expect(err).ToNot(HaveOccurred())

				transformedSecret := `---
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace`
				updatedFile, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())
				Expect(string(updatedFile)).To(ContainSubstring(transformedSecret))
				Expect(string(updatedFile)).ToNot(ContainSubstring(originalSecretContents))
				Expect(string(updatedFile)).To(ContainSubstring(wrongApiVersionForType))
			})
			It("updates only secrets with the proper labels if a mix of labeled and unlabeled secrets are present ", func() {
				var data = make(map[string][]byte)
				data["cert.pem"] = []byte(validPublicKey)
				clientset = fake.NewSimpleClientset(&v1.SecretList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Secret{{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret1",
							Namespace: namespace,
							Labels:    validLabels,
						},
						Data: data,
					}},
				})

				unlabeledSecretContents := `---
apiVersion: bitnami.com/v1alpha1
kind: Secret
metadata:
  creationTimestamp: null
  name: test-unlabeled-secret
  namespace: test-namespace
`
				originalSecretContents := `---
apiVersion: v1
kind: Secret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace
  labels:
    kots.io/buildphase: secret
    kots.io/secrettype: sealedsecrets`

				_, err := tmpFile.WriteString(unlabeledSecretContents)
				Expect(err).ToNot(HaveOccurred())
				_, err = tmpFile.WriteString(originalSecretContents)
				Expect(err).ToNot(HaveOccurred())

				err = secrets.ReplaceSecretsInPath(tmpArchiveDir, clientset)
				Expect(err).ToNot(HaveOccurred())

				transformedSecret := `---
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test-secret-1
  namespace: test-namespace`

				undesiredExtraWhitespace := `---

`
				updatedFile, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())
				Expect(string(updatedFile)).To(ContainSubstring(transformedSecret))
				Expect(string(updatedFile)).ToNot(ContainSubstring(originalSecretContents))
				Expect(string(updatedFile)).To(ContainSubstring(unlabeledSecretContents))
				Expect(string(updatedFile)).ToNot(ContainSubstring(undesiredExtraWhitespace))
			})
		})
	})
})

func writeSecret(apiVersion string, kind string, namespace string, dataOverride bool, labelOverride bool, tmpFile *os.File, numSecrets int) (error, string) {
	var data = ""
	if dataOverride {
		data = `data:
  invalid-cert-key: cHVibGljIGtleQo=`
	}

	var labels = `  labels:
    kots.io/buildphase: secret
    kots.io/secrettype: sealedsecrets`
	if labelOverride {
		labels = ""
	}

	var namespaceBlock = ""
	if namespace != "" {
		namespaceBlock = fmt.Sprintf(`  namespace: %s`, namespace)
	}

	var secret = ""
	for i := 0; i < numSecrets; i++ {
		secret = secret + fmt.Sprintf(`---
apiVersion: %s
kind: %s
metadata:
  name: test-secret-%d
%s
%s
%s
`, apiVersion, kind, i, namespaceBlock, labels, data)
	}

	_, err := tmpFile.WriteString(secret)

	return err, secret
}

var validPublicKey = "-----BEGIN CERTIFICATE-----\nMIICGzCCAYQCCQDuU+xTyPfxbjANBgkqhkiG9w0BAQsFADBRMQswCQYDVQQGEwJ1\nczELMAkGA1UECAwCb3IxETAPBgNVBAcMCHBvcnRsYW5kMRMwEQYDVQQKDApyZXBs\naWNhdGVkMQ0wCwYDVQQDDAR0ZXN0MCAXDTIyMDQyODE2MjIzMloYDzMwMjEwODI5\nMTYyMjMyWjBRMQswCQYDVQQGEwJ1czELMAkGA1UECAwCb3IxETAPBgNVBAcMCHBv\ncnRsYW5kMRMwEQYDVQQKDApyZXBsaWNhdGVkMQ0wCwYDVQQDDAR0ZXN0MIGfMA0G\nCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDXISiW5u8T7wznpVulmSEi42iszslO/Bym\n/2tVSO2XnD134eib8CZ4MjO5lXl8V8b5Rh6BeXsjsGRozKGuZ/up2EZ8wEVonvFH\nGaBQuCpx0LDEaXypZhMNDoV/zAJO0Ljf8RF4oSi1baAo/eY4V7ghTkxDYxhLoXzC\nAQh68OjOGQIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAJDTjqs8DyBd15IZ0XR8pMmp\n84R5OJ6SYTL0oga+fVgDMe8gEXxxf3g5nh/4PrcuJoP+oCXrNpOCuT5wr7ULbWNc\nIdVQsx42GSDJ2m6QABNOQncYLxXnDtNTuFQsKa+j5X8LEjdVfFc/ViW9k8VMLHS8\n++DdX288zaQ/2/ywPFxV\n-----END CERTIFICATE-----"

func generateInvalidPubKey() []byte {
	bitSize := 4096

	// Generate RSA key.
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		panic(err)
	}

	// Extract public component.
	pub := key.Public()

	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)

	return pubPEM
}
