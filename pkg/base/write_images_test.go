package base

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

func Test_RewriteImages(t *testing.T) {
	tests := []struct {
		name              string
		baseDir           string
		appSlug           string
		processOptions    RewriteImageOptions
		wantProcessResult RewriteImagesResult
		findOptions       FindPrivateImagesOptions
		wantFindResult    FindPrivateImagesResult
	}{
		{
			name:    "all unique",
			baseDir: "./testdata/base-specs",
			appSlug: "test-app-slug",
			processOptions: RewriteImageOptions{
				BaseDir: "./testdata/base-specs",
				SourceRegistry: registrytypes.RegistryOptions{
					Endpoint:      "registry.replicated.com",
					ProxyEndpoint: "proxy.replicated.com",
					Username:      "test-license-id",
					Password:      "test-license-id",
				},
				KotsKinds: &kotsutil.KotsKinds{
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
				CopyImages: false,
				AppSlug:    "test-app-slug",
				DestRegistry: registrytypes.RegistryOptions{
					Endpoint:  "ttl.sh",
					Namespace: "testing-ns",
					Username:  "testing-user-name",
					Password:  "testing-password",
				},
			},
			wantProcessResult: RewriteImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "busybox",
						NewName: "ttl.sh/testing-ns/busybox",
						NewTag:  "latest",
					},
					{
						Name:    "docker.io/library/busybox",
						NewName: "ttl.sh/testing-ns/busybox",
						NewTag:  "latest",
					},
					{
						Name:    "library/busybox",
						NewName: "ttl.sh/testing-ns/busybox",
						NewTag:  "latest",
					},
					{
						Name:    "docker.io/busybox",
						NewName: "ttl.sh/testing-ns/busybox",
						NewTag:  "latest",
					},
					{
						Name:    "registry.replicated.com/appslug/image",
						NewName: "ttl.sh/testing-ns/image",
						NewTag:  "version",
					},
					{
						Name:    "quay.io/replicatedcom/qa-kots-1",
						NewName: "ttl.sh/testing-ns/qa-kots-1",
						NewTag:  "alpine-3.5",
					},
					{
						Name:    "quay.io/replicatedcom/qa-kots-2",
						NewName: "ttl.sh/testing-ns/qa-kots-2",
						NewTag:  "alpine-3.4",
					},
					{
						Name:    "quay.io/replicatedcom/qa-kots-3",
						NewName: "ttl.sh/testing-ns/qa-kots-3",
						NewTag:  "alpine-3.6",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "ttl.sh/testing-ns/someimage",
						Digest:  "sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					},
					{
						Name:    "nginx",
						NewName: "ttl.sh/testing-ns/nginx",
						NewTag:  "1",
					},
					{
						Name:    "docker.io/library/nginx",
						NewName: "ttl.sh/testing-ns/nginx",
						NewTag:  "1",
					},
					{
						Name:    "library/nginx",
						NewName: "ttl.sh/testing-ns/nginx",
						NewTag:  "1",
					},
					{
						Name:    "docker.io/nginx",
						NewName: "ttl.sh/testing-ns/nginx",
						NewTag:  "1",
					},
					{
						Name:    "redis",
						NewName: "ttl.sh/testing-ns/redis",
						Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
					},
					{
						Name:    "docker.io/library/redis",
						NewName: "ttl.sh/testing-ns/redis",
						Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
					},
					{
						Name:    "library/redis",
						NewName: "ttl.sh/testing-ns/redis",
						Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
					},
					{
						Name:    "docker.io/redis",
						NewName: "ttl.sh/testing-ns/redis",
						Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "busybox",
						IsPrivate: false,
					},
					{
						Image:     "redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
						IsPrivate: false,
					},
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-3:alpine-3.6",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
						IsPrivate: true,
					},
					{
						Image:     "nginx:1",
						IsPrivate: false,
					},
				},
			},

			findOptions: FindPrivateImagesOptions{
				BaseDir: "./testdata/base-specs",
				AppSlug: "test-app-slug",
				ReplicatedRegistry: registrytypes.RegistryOptions{
					Endpoint:      "registry.replicated.com",
					ProxyEndpoint: "proxy.replicated.com",
					Username:      "test-license-id",
					Password:      "test-license-id",
				},
				Installation:     &kotsv1beta1.Installation{},
				AllImagesPrivate: false,
			},
			wantFindResult: FindPrivateImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "quay.io/replicatedcom/qa-kots-3",
						NewName: "proxy.replicated.com/proxy/test-app-slug/quay.io/replicatedcom/qa-kots-3",
						NewTag:  "alpine-3.6",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "proxy.replicated.com/proxy/test-app-slug/quay.io/replicatedcom/someimage",
						Digest:  "sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/qa-kots-3:alpine-3.6",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
						IsPrivate: true,
					},
					{
						Image:     "redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
						IsPrivate: false,
					},
					{
						Image:     "nginx:1",
						IsPrivate: false,
					},
					{
						Image:     "busybox",
						IsPrivate: false,
					},
				},
			},
		},
		{
			name:    "replicated registry with custom domains configured should rewrite replicated images and not custom domain images",
			baseDir: "./testdata/replicated-registry",
			appSlug: "test-app-slug",
			processOptions: RewriteImageOptions{
				BaseDir: "./testdata/replicated-registry",
				SourceRegistry: registrytypes.RegistryOptions{
					Endpoint:         "my-registry.example.com",
					ProxyEndpoint:    "my-proxy.example.com",
					UpstreamEndpoint: "registry.replicated.com",
					Username:         "test-license-id",
					Password:         "test-license-id",
				},
				KotsKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AdditionalImages: []string{},
						},
					},
				},
				CopyImages: false,
				AppSlug:    "test-app-slug",
				DestRegistry: registrytypes.RegistryOptions{
					Endpoint:  "ttl.sh",
					Namespace: "testing-ns",
					Username:  "testing-user-name",
					Password:  "testing-password",
				},
			},
			wantProcessResult: RewriteImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "registry.replicated.com/appslug/image",
						NewName: "ttl.sh/testing-ns/image",
						NewTag:  "version",
					},
					{
						Name:    "my-registry.example.com/appslug/some-other-image",
						NewName: "ttl.sh/testing-ns/some-other-image",
						NewTag:  "version",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "ttl.sh/testing-ns/someimage",
						NewTag:  "1",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "my-registry.example.com/appslug/some-other-image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1",
						IsPrivate: true,
					},
				},
			},

			findOptions: FindPrivateImagesOptions{
				BaseDir: "./testdata/replicated-registry",
				AppSlug: "test-app-slug",
				ReplicatedRegistry: registrytypes.RegistryOptions{
					Endpoint:         "my-registry.example.com",
					ProxyEndpoint:    "my-proxy.example.com",
					UpstreamEndpoint: "registry.replicated.com",
					Username:         "test-license-id",
					Password:         "test-license-id",
				},
				Installation:     &kotsv1beta1.Installation{},
				AllImagesPrivate: false,
			},
			wantFindResult: FindPrivateImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "registry.replicated.com/appslug/image",
						NewName: "my-registry.example.com/appslug/image",
						NewTag:  "version",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "my-proxy.example.com/proxy/test-app-slug/quay.io/replicatedcom/someimage",
						NewTag:  "1",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "my-registry.example.com/appslug/some-other-image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1",
						IsPrivate: true,
					},
				},
			},
		},
		{
			name:    "replicated registry without custom domains should not rewrite replicated registry images",
			baseDir: "./testdata/replicated-registry",
			appSlug: "test-app-slug",
			processOptions: RewriteImageOptions{
				BaseDir: "./testdata/replicated-registry",
				SourceRegistry: registrytypes.RegistryOptions{
					Endpoint:         "registry.replicated.com",
					ProxyEndpoint:    "proxy.replicated.com",
					UpstreamEndpoint: "registry.replicated.com",
					Username:         "test-license-id",
					Password:         "test-license-id",
				},
				KotsKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AdditionalImages: []string{},
						},
					},
				},
				CopyImages: false,
				AppSlug:    "test-app-slug",
				DestRegistry: registrytypes.RegistryOptions{
					Endpoint:  "ttl.sh",
					Namespace: "testing-ns",
					Username:  "testing-user-name",
					Password:  "testing-password",
				},
			},
			wantProcessResult: RewriteImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "registry.replicated.com/appslug/image",
						NewName: "ttl.sh/testing-ns/image",
						NewTag:  "version",
					},
					{
						Name:    "my-registry.example.com/appslug/some-other-image",
						NewName: "ttl.sh/testing-ns/some-other-image",
						NewTag:  "version",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "ttl.sh/testing-ns/someimage",
						NewTag:  "1",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "my-registry.example.com/appslug/some-other-image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1",
						IsPrivate: true,
					},
				},
			},

			findOptions: FindPrivateImagesOptions{
				BaseDir: "./testdata/replicated-registry",
				AppSlug: "test-app-slug",
				ReplicatedRegistry: registrytypes.RegistryOptions{
					Endpoint:      "registry.replicated.com",
					ProxyEndpoint: "proxy.replicated.com",
					Username:      "test-license-id",
					Password:      "test-license-id",
				},
				Installation:     &kotsv1beta1.Installation{},
				AllImagesPrivate: false,
			},
			wantFindResult: FindPrivateImagesResult{
				Images: []kustomizeimage.Image{
					{
						Name:    "my-registry.example.com/appslug/some-other-image",
						NewName: "proxy.replicated.com/proxy/test-app-slug/my-registry.example.com/appslug/some-other-image",
						NewTag:  "version",
					},
					{
						Name:    "quay.io/replicatedcom/someimage",
						NewName: "proxy.replicated.com/proxy/test-app-slug/quay.io/replicatedcom/someimage",
						NewTag:  "1",
					},
				},
				CheckedImages: []kotsv1beta1.InstallationImage{
					{
						Image:     "registry.replicated.com/appslug/image:version",
						IsPrivate: true,
					},
					{
						Image:     "my-registry.example.com/appslug/some-other-image:version",
						IsPrivate: true,
					},
					{
						Image:     "quay.io/replicatedcom/someimage:1",
						IsPrivate: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			gotUpstreamResult, err := RewriteImages(test.processOptions)
			req.NoError(err)

			assert.ElementsMatch(t, test.wantProcessResult.Images, gotUpstreamResult.Images)
			assert.ElementsMatch(t, test.wantProcessResult.CheckedImages, gotUpstreamResult.CheckedImages)

			test.findOptions.KotsKindsImages = kotsutil.GetImagesFromKotsKinds(test.processOptions.KotsKinds)
			gotFindResult, err := FindPrivateImages(test.findOptions)
			req.NoError(err)

			wantDocs, err := loadDocs(test.baseDir)
			req.NoError(err)

			assert.ElementsMatch(t, test.wantFindResult.Images, gotFindResult.Images)
			assert.ElementsMatch(t, wantDocs, gotFindResult.Docs)
			assert.ElementsMatch(t, test.wantFindResult.CheckedImages, gotFindResult.CheckedImages)
		})
	}

}

func loadDocs(basePath string) ([]k8sdoc.K8sDoc, error) {
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		return nil, errors.Wrap(err, "read base dir")
	}

	docs := []k8sdoc.K8sDoc{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(basePath, file.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "read file")
		}

		doc, err := k8sdoc.ParseYAML(content)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}
