package registry

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetRegistryProxyInfo(t *testing.T) {
	customProxy, customRegistry := "custom.proxy.com", "custom.registry.com"
	type args struct {
		license      *kotsv1beta1.License
		installation *kotsv1beta1.Installation
		app          *kotsv1beta1.Application
	}
	tests := []struct {
		name string
		args args
		env  map[string]string
		want *RegistryProxyInfo
	}{
		{
			name: "GetRegistryProxyInfo returns default proxy info when installation, app, and license are nil",
			args: args{
				license:      nil,
				installation: nil,
				app:          nil,
			},
			want: &RegistryProxyInfo{
				Registry: "registry.replicated.com",
				Proxy:    "proxy.replicated.com",
				Upstream: "registry.replicated.com",
			},
		},
		{
			name: "GetRegistryProxyInfo returns custom registry hostnames when installation has registry settings",
			args: args{
				license: nil,
				installation: &kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						ReplicatedProxyDomain:    customProxy,
						ReplicatedRegistryDomain: customRegistry,
					},
				},
			},
			want: &RegistryProxyInfo{
				Registry: customRegistry,
				Proxy:    customProxy,
				Upstream: "registry.replicated.com",
			},
		},
		{
			name: "GetRegistryProxyInfo returns custom registry hostnames when app has registry settings",
			args: args{
				license: nil,
				app: &kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ProxyRegistryDomain:      customProxy,
						ReplicatedRegistryDomain: customRegistry,
					},
				},
			},
			want: &RegistryProxyInfo{
				Registry: customRegistry,
				Proxy:    customProxy,
				Upstream: "registry.replicated.com",
			},
		},
		{
			name: "GetRegistryProxyInfo returns custom registry hostnames from installation when both installation and app has registry settings",
			args: args{
				license: nil,
				installation: &kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						ReplicatedProxyDomain:    customProxy,
						ReplicatedRegistryDomain: customRegistry,
					},
				},
				app: &kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ProxyRegistryDomain:      "random.proxy.com",
						ReplicatedRegistryDomain: "random.registry.com",
					},
				},
			},
			want: &RegistryProxyInfo{
				Registry: customRegistry,
				Proxy:    customProxy,
				Upstream: "registry.replicated.com",
			},
		},
		{
			name: "GetRegistryProxyInfo returns embedded cluster registry when EMBEDDED_CLUSTER is set",
			args: args{
				license:      nil,
				installation: nil,
				app:          nil,
			},
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":        "123",
				"REPLICATED_REGISTRY_DOMAIN": "localhost:30000",
				"PROXY_REGISTRY_DOMAIN":      "localhost:30001",
			},
			want: &RegistryProxyInfo{
				Registry: "localhost:30000",
				Proxy:    "localhost:30001",
				Upstream: "registry.replicated.com",
			},
		},
		{
			name: "GetRegistryProxyInfo does not use installation settings for embedded cluster",
			args: args{
				license: nil,
				installation: &kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						ReplicatedProxyDomain:    customProxy,
						ReplicatedRegistryDomain: customRegistry,
					},
				},
				app: nil,
			},
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID":        "123",
				"REPLICATED_REGISTRY_DOMAIN": "localhost:30000",
				"PROXY_REGISTRY_DOMAIN":      "localhost:30001",
			},
			want: &RegistryProxyInfo{
				Registry: "localhost:30000",
				Proxy:    "localhost:30001",
				Upstream: "registry.replicated.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := GetRegistryProxyInfo(tt.args.license, tt.args.installation, tt.args.app)
			if err != nil {
				t.Errorf("GetRegistryProxyInfo() error = %v", err)
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRegistryProxyInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRegistryProxyEndpointFromKotsInstallation(t *testing.T) {
	customProxy, customRegistry := "custom.proxy.com", "custom.registry.com"
	type args struct {
		kotsInstallation *kotsv1beta1.Installation
	}
	tests := []struct {
		name                 string
		args                 args
		wantProxyEndpoint    string
		wantRegistryEndpoint string
	}{
		{
			name:                 "getRegistryProxyEndpointFromKotsInstallation returns nil when kotsInstallation is nil",
			args:                 args{kotsInstallation: nil},
			wantProxyEndpoint:    "",
			wantRegistryEndpoint: "",
		},
		{
			name:                 "getRegistryProxyEndpointFromKotsInstallation returns nil when kotsInstallation is not nil but has no registry settings",
			args:                 args{kotsInstallation: &kotsv1beta1.Installation{}},
			wantProxyEndpoint:    "",
			wantRegistryEndpoint: "",
		},
		{
			name: "getRegistryProxyEndpointFromKotsInstallation returns endpoints nil when kotsInstallation and registry settings are not nil",
			args: args{kotsInstallation: &kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					ReplicatedProxyDomain:    customProxy,
					ReplicatedRegistryDomain: customRegistry,
				},
			},
			},
			wantProxyEndpoint:    customProxy,
			wantRegistryEndpoint: customRegistry,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProxyEndpoint, gotRegistryEndpoint := getRegistryProxyEndpointFromKotsInstallation(tt.args.kotsInstallation)
			if gotProxyEndpoint != tt.wantProxyEndpoint {
				t.Errorf("getRegistryProxyEndpointFromKotsInstallation() gotProxyEndpoint = %v, want %v", gotProxyEndpoint, tt.wantProxyEndpoint)
			}
			if gotRegistryEndpoint != tt.wantRegistryEndpoint {
				t.Errorf("getRegistryProxyEndpointFromKotsInstallation() gotRegistryEndpoint = %v, want %v", gotRegistryEndpoint, tt.wantRegistryEndpoint)
			}
		})
	}
}

func Test_getRegistryProxyEndpointFromKotsApplication(t *testing.T) {
	customProxy, customRegistry := "custom.proxy.com", "custom.registry.com"
	type args struct {
		kotsApplication *kotsv1beta1.Application
	}
	tests := []struct {
		name                 string
		args                 args
		wantProxyEndpoint    string
		wantRegistryEndpoint string
	}{
		{
			name:                 "getRegistryProxyEndpointFromKotsApplication returns nil when kotsApplication is nil",
			args:                 args{kotsApplication: nil},
			wantProxyEndpoint:    "",
			wantRegistryEndpoint: "",
		},
		{
			name:                 "getRegistryProxyEndpointFromKotsApplication returns nil when kotsApplication is not nil but has no registry settings",
			args:                 args{kotsApplication: &kotsv1beta1.Application{}},
			wantProxyEndpoint:    "",
			wantRegistryEndpoint: "",
		},
		{
			name: "getRegistryProxyEndpointFromKotsApplication returns endpoints nil when kotsApplication and registry settings are not nil",
			args: args{kotsApplication: &kotsv1beta1.Application{
				Spec: kotsv1beta1.ApplicationSpec{
					ProxyRegistryDomain:      customProxy,
					ReplicatedRegistryDomain: customRegistry,
				},
			},
			},
			wantProxyEndpoint:    customProxy,
			wantRegistryEndpoint: customRegistry,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProxyEndpoint, gotRegistryEndpoint := getRegistryProxyEndpointFromKotsApplication(tt.args.kotsApplication)
			if gotProxyEndpoint != tt.wantProxyEndpoint {
				t.Errorf("getRegistryProxyEndpointFromKotsApplication() gotProxyEndpoint = %v, want %v", gotProxyEndpoint, tt.wantProxyEndpoint)
			}
			if gotRegistryEndpoint != tt.wantRegistryEndpoint {
				t.Errorf("getRegistryProxyEndpointFromKotsApplication() gotRegistryEndpoint = %v, want %v", gotRegistryEndpoint, tt.wantRegistryEndpoint)
			}
		})
	}
}

func Test_getRegistryProxyInfoFromLicense(t *testing.T) {
	tests := []struct {
		name    string
		license *kotsv1beta1.License
		want    *RegistryProxyInfo
	}{
		{
			name:    "ProxyEndpointFromLicense with nil license parameter returns default proxy info",
			license: nil,
			want: &RegistryProxyInfo{
				Registry: "registry.replicated.com",
				Proxy:    "proxy.replicated.com",
			},
		}, {
			name:    "ProxyEndpointFromLicense with invalid url parameter for spec endpoint returns default proxy info",
			license: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{Endpoint: "<>invalidurl>>"}},
			want: &RegistryProxyInfo{
				Registry: "registry.replicated.com",
				Proxy:    "proxy.replicated.com",
			},
		},
		{
			name:    "ProxyEndpointFromLicense with license parameter containing staging spec endpoint returns staging proxy info",
			license: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{Endpoint: "protocol://user:pwd@staging.replicated.app"}},
			want: &RegistryProxyInfo{
				Registry: "registry.staging.replicated.com",
				Proxy:    "proxy.staging.replicated.com",
			},
		}, {
			name:    "ProxyEndpointFromLicense with license parameter containing a dev (okteto) endpoint returns the same naemspace proxy info",
			license: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{Endpoint: "protocol://user:pwd@replicated-app-user1.okteto.repldev.com"}},
			want: &RegistryProxyInfo{
				Registry: "vendor-registry-v2-user1.okteto.repldev.com",
				Proxy:    "registry-proxy-user1.okteto.repldev.com",
			},
		}, {
			name:    "ProxyEndpointFromLicense returns default info when url parsing fails",
			license: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{Endpoint: "protocol://use<<>>r:pwd@replicated-app"}},
			want: &RegistryProxyInfo{
				Registry: "registry.replicated.com",
				Proxy:    "proxy.replicated.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getRegistryProxyInfoFromLicense(tt.license)
			if res.Registry != tt.want.Registry || res.Proxy != tt.want.Proxy {
				t.Errorf("ProxyEndpointFromLicense() = %v, want %v", res, tt.want)
			}
		})
	}
}

func Test_ToSlice(t *testing.T) {
	tests := []struct {
		name      string
		proxyInfo *RegistryProxyInfo
		want      []string
	}{
		{
			name:      "ToSlice returns slice with 2 values, the proxy and the registry",
			proxyInfo: &RegistryProxyInfo{Proxy: "myProxy", Registry: "myRegistry"},
			want:      []string{"myProxy", "myRegistry"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.proxyInfo.ToSlice()
			if res[0] != tt.want[0] || res[1] != tt.want[1] {
				t.Errorf("ToSlice() = %v, want %v", res, tt.want)
			}
		})
	}
}

func Test_SecretNameFromPrefix(t *testing.T) {
	tests := []struct {
		name       string
		namePrefix string
		want       string
	}{
		{
			name:       "SecretNameFromPrefix returns empty string when prefix is empty",
			namePrefix: "",
			want:       "",
		}, {
			name:       "SecretNameFromPrefix returns string with nameprefix as prefix and -registry as suffix",
			namePrefix: "myPrefix",
			want:       "myPrefix-registry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := SecretNameFromPrefix(tt.namePrefix); res != tt.want {
				t.Errorf("SecretNameFromPrefix() = %v, want %v", res, tt.want)
			}
		})
	}
}

func TestPullSecretForRegistries(t *testing.T) {
	type args struct {
		registries   []string
		username     string
		password     string
		appNamespace string
		namePrefix   string
	}
	tests := []struct {
		name string
		args args
		env  map[string]string
		want ImagePullSecrets
	}{
		{
			name: "not embedded-cluster",
			args: args{
				registries:   []string{"registry.replicated.com"},
				username:     "username",
				password:     "password",
				appNamespace: "test-namespace",
				namePrefix:   "my-app",
			},
			want: ImagePullSecrets{
				AdminConsoleSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-replicated-registry",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"kots.io/creation-phase": "-9999",
							"helm.sh/hook":           "pre-install,pre-upgrade",
							"helm.sh/hook-weight":    "-9999",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(fmt.Sprintf(`{"auths":{"registry.replicated.com":{"auth":"%s"}}}`, base64.StdEncoding.EncodeToString([]byte("username:password")))),
					},
				},
				AppSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-app-registry",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"kots.io/creation-phase": "-9999",
							"helm.sh/hook":           "pre-install,pre-upgrade",
							"helm.sh/hook-weight":    "-9999",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(fmt.Sprintf(`{"auths":{"registry.replicated.com":{"auth":"%s"}}}`, base64.StdEncoding.EncodeToString([]byte("username:password")))),
					},
				},
			},
		},
		{
			name: "embedded-cluster",
			args: args{
				registries:   []string{"registry.replicated.com"},
				username:     "username",
				password:     "password",
				appNamespace: "test-namespace",
				namePrefix:   "my-app",
			},
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID": "embedded-cluster-id",
			},
			want: ImagePullSecrets{
				AdminConsoleSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-replicated-registry",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"kots.io/creation-phase": "-9999",
							"helm.sh/hook":           "pre-install,pre-upgrade",
							"helm.sh/hook-weight":    "-9999",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(fmt.Sprintf(`{"auths":{"registry.replicated.com":{"auth":"%s"}}}`, base64.StdEncoding.EncodeToString([]byte("username:password")))),
					},
				},
				AppSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-app-registry",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"kots.io/creation-phase": "-9999",
							"helm.sh/hook":           "pre-install,pre-upgrade",
							"helm.sh/hook-weight":    "-9999",
						},
						Labels: map[string]string{
							"replicated.com/disaster-recovery":       "infra",
							"replicated.com/disaster-recovery-chart": "admin-console",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(fmt.Sprintf(`{"auths":{"registry.replicated.com":{"auth":"%s"}}}`, base64.StdEncoding.EncodeToString([]byte("username:password")))),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := PullSecretForRegistries(tt.args.registries, tt.args.username, tt.args.password, tt.args.appNamespace, tt.args.namePrefix)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func TestGetDockerHubPullSecret(t *testing.T) {
	type args struct {
		clientset    kubernetes.Interface
		namespace    string
		appNamespace string
		namePrefix   string
	}
	tests := []struct {
		name string
		args args
		env  map[string]string
		want *corev1.Secret
	}{
		{
			name: "name prefix is empty",
			args: args{
				clientset:    fake.NewSimpleClientset(),
				namespace:    "test-namespace",
				appNamespace: "test-namespace",
				namePrefix:   "",
			},
			want: nil,
		},
		{
			name: "dockerhub secret does not exist",
			args: args{
				clientset:    fake.NewSimpleClientset(),
				namespace:    "test-namespace",
				appNamespace: "test-namespace",
				namePrefix:   "my-app",
			},
			want: nil,
		},
		{
			name: "dockerhub secret exists - not embedded-cluster",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-dockerhub",
						Namespace: "test-namespace",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`this would be a base64 encoded docker config json`),
					},
				}),
				namespace:    "test-namespace",
				appNamespace: "test-namespace",
				namePrefix:   "my-app",
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app-kotsadm-dockerhub",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"kots.io/creation-phase": "-9999",
						"helm.sh/hook":           "pre-install,pre-upgrade",
						"helm.sh/hook-weight":    "-9999",
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					".dockerconfigjson": []byte(`this would be a base64 encoded docker config json`),
				},
			},
		},
		{
			name: "dockerhub secret exists - embedded-cluster",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-dockerhub",
						Namespace: "test-namespace",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`this would be a base64 encoded docker config json`),
					},
				}),
				namespace:    "test-namespace",
				appNamespace: "test-namespace",
				namePrefix:   "my-app",
			},
			env: map[string]string{
				"EMBEDDED_CLUSTER_ID": "embedded-cluster-id",
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app-kotsadm-dockerhub",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"kots.io/creation-phase": "-9999",
						"helm.sh/hook":           "pre-install,pre-upgrade",
						"helm.sh/hook-weight":    "-9999",
					},
					Labels: map[string]string{
						"replicated.com/disaster-recovery":       "infra",
						"replicated.com/disaster-recovery-chart": "admin-console",
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					".dockerconfigjson": []byte(`this would be a base64 encoded docker config json`),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := GetDockerHubPullSecret(tt.args.clientset, tt.args.namespace, tt.args.appNamespace, tt.args.namePrefix)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
