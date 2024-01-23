package base

import (
	"testing"

	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func Test_RewriteImages(t *testing.T) {
	type args struct {
		images              []string
		destinationRegistry registrytypes.RegistryOptions
	}

	tests := []struct {
		name       string
		args       args
		wantResult []kustomizetypes.Image
	}{
		{
			name: "basic",
			args: args{
				images: []string{
					"quay.io/replicatedcom/qa-kots-3:alpine-3.6",
					"quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					"redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
					"registry.replicated.com/appslug/image:version",
					"quay.io/replicatedcom/qa-kots-1:alpine-3.5",
					"nginx:1",
					"quay.io/replicatedcom/qa-kots-2:alpine-3.4",
					"testing.registry.com:5000/testing-ns/random-image:1",
					"busybox",
				},
				destinationRegistry: registrytypes.RegistryOptions{
					Endpoint:  "testing.registry.com:5000",
					Namespace: "testing-ns",
					Username:  "testing-user-name",
					Password:  "testing-password",
				},
			},
			wantResult: []kustomizetypes.Image{
				{
					Name:    "busybox",
					NewName: "testing.registry.com:5000/testing-ns/busybox",
					NewTag:  "latest",
				},
				{
					Name:    "docker.io/library/busybox",
					NewName: "testing.registry.com:5000/testing-ns/busybox",
					NewTag:  "latest",
				},
				{
					Name:    "library/busybox",
					NewName: "testing.registry.com:5000/testing-ns/busybox",
					NewTag:  "latest",
				},
				{
					Name:    "docker.io/busybox",
					NewName: "testing.registry.com:5000/testing-ns/busybox",
					NewTag:  "latest",
				},
				{
					Name:    "registry.replicated.com/appslug/image",
					NewName: "testing.registry.com:5000/testing-ns/image",
					NewTag:  "version",
				},
				{
					Name:    "quay.io/replicatedcom/qa-kots-1",
					NewName: "testing.registry.com:5000/testing-ns/qa-kots-1",
					NewTag:  "alpine-3.5",
				},
				{
					Name:    "quay.io/replicatedcom/qa-kots-2",
					NewName: "testing.registry.com:5000/testing-ns/qa-kots-2",
					NewTag:  "alpine-3.4",
				},
				{
					Name:    "quay.io/replicatedcom/qa-kots-3",
					NewName: "testing.registry.com:5000/testing-ns/qa-kots-3",
					NewTag:  "alpine-3.6",
				},
				{
					Name:    "quay.io/replicatedcom/someimage",
					NewName: "testing.registry.com:5000/testing-ns/someimage",
					Digest:  "sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
				},
				{
					Name:    "nginx",
					NewName: "testing.registry.com:5000/testing-ns/nginx",
					NewTag:  "1",
				},
				{
					Name:    "docker.io/library/nginx",
					NewName: "testing.registry.com:5000/testing-ns/nginx",
					NewTag:  "1",
				},
				{
					Name:    "library/nginx",
					NewName: "testing.registry.com:5000/testing-ns/nginx",
					NewTag:  "1",
				},
				{
					Name:    "docker.io/nginx",
					NewName: "testing.registry.com:5000/testing-ns/nginx",
					NewTag:  "1",
				},
				{
					Name:    "redis",
					NewName: "testing.registry.com:5000/testing-ns/redis",
					Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "testing.registry.com:5000/testing-ns/redis",
					Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
				},
				{
					Name:    "library/redis",
					NewName: "testing.registry.com:5000/testing-ns/redis",
					Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
				},
				{
					Name:    "docker.io/redis",
					NewName: "testing.registry.com:5000/testing-ns/redis",
					Digest:  "sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
				},
				{
					Name:    "testing.registry.com:5000/testing-ns/random-image",
					NewName: "testing.registry.com:5000/testing-ns/random-image",
					NewTag:  "1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			gotResult, err := RewriteImages(test.args.images, test.args.destinationRegistry)
			req.NoError(err)

			assert.ElementsMatch(t, test.wantResult, gotResult)
		})
	}
}

func Test_RewritePrivateImages(t *testing.T) {
	type args struct {
		images    []string
		kotsKinds *kotsutil.KotsKinds
	}

	tests := []struct {
		name       string
		args       args
		wantResult []kustomizetypes.Image
	}{
		{
			name: "basic",
			args: args{
				images: []string{
					"quay.io/replicatedcom/qa-kots-3:alpine-3.6",
					"quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					"redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
				},
				kotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							AppSlug: "test-app-slug",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							KnownImages: []kotsv1beta1.InstallationImage{
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
							},
						},
					},
				},
			},
			wantResult: []kustomizetypes.Image{
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
		},
		{
			name: "replicated registry with custom domains configured should rewrite replicated images and not custom domain images",
			args: args{
				images: []string{
					"registry.replicated.com/appslug/image:version",
					"my-registry.example.com/appslug/some-other-image:version",
					"quay.io/replicatedcom/someimage:1",
				},
				kotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							AppSlug: "test-app-slug",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ReplicatedRegistryDomain: "my-registry.example.com",
							ReplicatedProxyDomain:    "my-proxy.example.com",
							KnownImages: []kotsv1beta1.InstallationImage{
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
				},
			},
			wantResult: []kustomizetypes.Image{
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
		},
		{
			name: "replicated registry without custom domains should not rewrite replicated registry images",
			args: args{
				images: []string{
					"registry.replicated.com/appslug/image:version",
					"my-registry.example.com/appslug/some-other-image:version",
					"quay.io/replicatedcom/someimage:1",
				},
				kotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							AppSlug: "test-app-slug",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							KnownImages: []kotsv1beta1.InstallationImage{
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
				},
			},
			wantResult: []kustomizetypes.Image{
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			gotResult, err := RewritePrivateImages(test.args.images, test.args.kotsKinds, test.args.kotsKinds.License)
			req.NoError(err)

			assert.ElementsMatch(t, test.wantResult, gotResult)
		})
	}
}
