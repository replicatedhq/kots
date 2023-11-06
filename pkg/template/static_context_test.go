package template

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestStaticContext_kubeSeal_badCert(t *testing.T) {
	req := require.New(t)

	ctx := StaticCtx{}

	cert := `-----BEGIN CERTIFICATE-----
SPA=
-----END CERTIFICATE-----`

	sealed, err := ctx.kubeSeal(cert, "default", "mysecret", "clear")
	req.Error(err, "invalid cert should return an error")
	req.Empty(sealed, "should return empty string with provided invalid cert")
}

func TestStaticContext_kubeSeal_goodCert(t *testing.T) {
	req := require.New(t)

	ctx := StaticCtx{}

	cert := `-----BEGIN CERTIFICATE-----
MIIErTCCApWgAwIBAgIQCDeIIck6VL8U9rDEP8ZECjANBgkqhkiG9w0BAQsFADAA
MB4XDTE5MDEyNTE1MTIzMFoXDTI5MDEyMjE1MTIzMFowADCCAiIwDQYJKoZIhvcN
AQEBBQADggIPADCCAgoCggIBALVuzNLJdwJ+cNkSINeMOmSgusRkkbZ5pqb7oeyw
l6CwoqE2pshplKwqs5jDjidkGyJ/TjIa8Ar1cOTY3GO+OVc5xNQoTrCWhFZA9v/Q
cY8aw91BV1Fha80gBgN4hghdiZnATLLBgT1pgCGnQhH8GLunQqSH+lV3TS0fnUGZ
ivUld2gU8bvjoBzdd0o8aW/5VDYlyOmP9MnMldOcQ4kFNpdAtnKOBAvoxv4C94v6
2aUo8j7XFpJpYJeECtlTATRvRC1pvDok3c5Mr7hgPy8Gsdatc6xvMkY2sZ0GBP4j
ta2k2wGT8pmXUFMlSKrBYoOZ5t/VjalMZBUUYcCnkcFrDrm5p7HW1upei+z5THkg
Y/H0foPj91vkB0BeL00QZOu30uTAVL+F/1Al0lg4VtTHiNMIruos91+F7SweSIXZ
TkvFPpPK3d2Jt55o/cuW5m/JxzNqMlpVD2Foc7OoWy0PodPy+rFWYgRBUiGShq/2
rC2UXCCisK/NL1ellW5p6K12tkBwoYLROe2IVPGVF7OCzLU8IbKHCzZ3E/O25CdL
7cxC62kvCDNJI+s+cpRDg1OZppir6NCbEPPlQbmRPS/8NN0HJKULPCiGcpvGHG4r
tWdkN8tc2wRyRQGAdxJr1CD0XLQO+Xo3f2wfvLiBwLpgqn33jfH3Ybnl9vU1jEYm
abi9AgMBAAGjIzAhMA4GA1UdDwEB/wQEAwIAATAPBgNVHRMBAf8EBTADAQH/MA0G
CSqGSIb3DQEBCwUAA4ICAQCTIwRIbCg+TB6/ZSzj1/52o/yEuqlOne2Og22B14pU
JiwfpWm5GgG0jOBNHZ7f3EL/bdB5lQJXtvcAx8SwBLk8g/1ieSCcm23kDgdkls+F
om0uiyXWTqJC+vktPhWyzFm3ltvHFNOcxo7eRXica7gNKbcFTQWmdkI+SwuBkBYp
gQjxAZUnpN8byAyuZwPTRWVNmUF21DarC6Ol5TDt4HBGFv5ItLVxVh40t10jghOg
xgr1i38dxZzaYQA+WdeDlJqbsu5tKGihEjBAgAMbhDe194y0ROjZJtdKG/gH7P0g
ttIrjatPvoVLeHiS1bRpLEwbSOYi2yecQcvHLlsKUedQ/168oLR94jnlS8jejRAH
WKwVf1QBx/UJWDW6SsunFuLMNUDUhhTfR6zOQNkD8AwIH7wsENIz0Denp/R0rsyJ
bUrDhgY9tB4JpDXYcquL+zkC7Ivc3XBFNCEp5H/9njoDXHhit6Dv2eTG3Tu05Uzf
Jrav2KaQ9pp8Oj28C1VTmTPMqZPpF7ZyfrFOHj/9wzOM5ecsQ4Ee91MyTkdyOB29
f+ipeUSsw5HLNxJpkAZYclBkN+ZnL35FsH4wmEyqZodj+dDgkIiSQq6p3QfQ8NUu
JgtOJZLvegJ7oeZsG8zHDP2BvmScSPJq6nLTD5i4P04EOIDNjH6GfNFKVwp9GSPT
yw==
-----END CERTIFICATE-----`

	sealed, err := ctx.kubeSeal(cert, "default", "mysecret", "clear")
	req.NoError(err, "kubeSeal should not return an error with a valid cert")
	req.NotEmpty(sealed, "should return a non empty encrypted secret")
}

func TestSprigRandom(t *testing.T) {
	req := require.New(t)

	builder := Builder{}
	builder.AddCtx(StaticCtx{})

	randAlphaNum, err := builder.String("{{repl randAlphaNum 50}}")

	req.NoError(err)
	req.Len(randAlphaNum, 50)
}

func validateAndClearCaCert(req *require.Assertions, builder Builder) {
	caCert, err := builder.String(`{{repl TLSCACert "my-ca" 365}}`)
	req.NoError(err)

	cert, err := getCert(caCert)
	req.NoError(err)
	req.NotZero(cert.KeyUsage & x509.KeyUsageCertSign)

	expected := caMap["my-ca"]
	req.Equal(expected.Cert, caCert)
	delete(caMap, "my-ca")
}

func TestTlsCaCert(t *testing.T) {
	req := require.New(t)

	builder := Builder{}
	builder.AddCtx(StaticCtx{})
	validateAndClearCaCert(req, builder)
}

func TestTlsCertFromCa(t *testing.T) {
	req := require.New(t)

	builder := Builder{}
	builder.AddCtx(StaticCtx{})

	cert, err := builder.String(`{{repl TLSCertFromCA "my-ca" "my-cert" "mine.example.com" nil nil 365}}`)
	req.NoError(err)

	certObj, err := getCert(cert)
	req.NoError(err)
	req.Equal("CN=mine.example.com", certObj.Subject.String())
	req.Equal("CN=my-ca", certObj.Issuer.String())

	expected := tlsMap["my-ca:my-cert:mine.example.com"]
	req.Equal("mine.example.com", expected.Cn)
	req.Equal(expected.Cert, cert)

	_, err = builder.String(`{{repl TLSKeyFromCA "my-ca" "my-cert" "mine.example.com" nil nil 365}}`)
	req.NoError(err)

	validateAndClearCaCert(req, builder)
}

func TestTlsKeyFromCa(t *testing.T) {
	req := require.New(t)

	builder := Builder{}
	builder.AddCtx(StaticCtx{})

	_, err := builder.String(`{{repl TLSKeyFromCA "my-ca" "my-cert" "mine.example.com" nil nil 365}}`)
	req.NoError(err)

	cert, err := builder.String(`{{repl TLSCertFromCA "my-ca" "my-cert" "mine.example.com" nil nil 365}}`)
	req.NoError(err)

	certObj, err := getCert(cert)
	req.NoError(err)
	req.Equal("CN=mine.example.com", certObj.Subject.String())
	req.Equal("CN=my-ca", certObj.Issuer.String())

	expected := tlsMap["my-ca:my-cert:mine.example.com"]
	req.Equal("mine.example.com", expected.Cn)
	req.Equal(expected.Cert, cert)
	delete(tlsMap, "my-ca:my-cert:mine.example.com")

	validateAndClearCaCert(req, builder)
}

func getCert(s string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM: %s", s)
	}

	return x509.ParseCertificate(block.Bytes)
}

func TestYamlEscape(t *testing.T) {
	req := require.New(t)

	allchars := ""
	for i := 0; i <= 255; i++ {
		allchars += string(byte(i))
	}

	allchars += "hello\nworld\nmultiple\nlines"

	encoded := StaticCtx{}.yamlEscape(allchars)
	req.Greater(len(encoded), len(allchars)) // the encoded version will be wrapped in quotes, and have escape characters

	decoded := ""
	err := yaml.Unmarshal([]byte(encoded), &decoded)
	req.NoError(err)
	req.Equal(allchars, decoded)

	exampleYamlTpl := `
abc:
  xyz: %s`
	exampleYaml := fmt.Sprintf(exampleYamlTpl, encoded)
	type xyz struct {
		XYZ string `yaml:"xyz"`
	}
	type abc struct {
		ABC xyz `yaml:"abc"`
	}

	abcTest := abc{}
	err = yaml.Unmarshal([]byte(exampleYaml), &abcTest)
	req.NoError(err)
	req.Equal(allchars, abcTest.ABC.XYZ)
}

func TestProxyEnvVars(t *testing.T) {
	req := require.New(t)

	t.Setenv("HTTPS_PROXY", "1.1.1.1")
	t.Setenv("HTTP_PROXY", "2.2.2.2")
	t.Setenv("NO_PROXY", "3.3.3.3")

	builder := Builder{}
	builder.AddCtx(StaticCtx{})

	httpsProxy, err := builder.String(`{{repl HTTPSProxy}}`)
	req.NoError(err)

	httpProxy, err := builder.String(`{{repl HTTPProxy}}`)
	req.NoError(err)

	noProxy, err := builder.String(`{{repl NoProxy}}`)
	req.NoError(err)

	req.Equal(httpsProxy, "1.1.1.1")
	req.Equal(httpProxy, "2.2.2.2")
	req.Equal(noProxy, "3.3.3.3")
}

func TestKubernetesVersion(t *testing.T) {
	clientset := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Error("couldn't convert Discovery() to *FakeDiscovery")
		return
	}

	wantK8sVersion := "v1.25.0"
	wantK8sMajorVersion := "1"
	wantK8sMinorVersion := "25"

	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: wantK8sVersion,
		Major:      wantK8sMajorVersion,
		Minor:      wantK8sMinorVersion,
	}

	ctx := StaticCtx{
		clientset: clientset,
	}

	if actualK8sVersion := ctx.kubernetesVersion(); actualK8sVersion != strings.TrimPrefix(wantK8sVersion, "v") {
		t.Errorf("expected kubernetes version to be %q, got %q", strings.TrimPrefix(wantK8sVersion, "v"), actualK8sVersion)
	}

	if actualK8sMajorVersion := ctx.kubernetesMajorVersion(); actualK8sMajorVersion != wantK8sMajorVersion {
		t.Errorf("expected kubernetes major version to be %q, got %q", wantK8sMajorVersion, actualK8sMajorVersion)
	}

	if actualK8sMinorVersion := ctx.kubernetesMinorVersion(); actualK8sMinorVersion != wantK8sMinorVersion {
		t.Errorf("expected kubernetes minor version to be %q, got %q", wantK8sMinorVersion, actualK8sMinorVersion)
	}
}

type mockClientsetForDistributionOpts struct {
	objects       []runtime.Object
	k8sVersion    string
	groupVersions []string
}

func mockClientsetForDistribution(opts *mockClientsetForDistributionOpts) kubernetes.Interface {
	clientset := fake.NewSimpleClientset(opts.objects...)
	resources := []*metav1.APIResourceList{}
	for _, groupVersion := range opts.groupVersions {
		resources = append(resources, &metav1.APIResourceList{
			GroupVersion: groupVersion,
		})
	}
	clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources = resources
	clientset.Discovery().(*discoveryfake.FakeDiscovery).FakedServerVersion = &version.Info{
		GitVersion: opts.k8sVersion,
	}
	return clientset
}

func TestDistribution(t *testing.T) {
	tests := []struct {
		name      string
		clientset kubernetes.Interface
		want      string
	}{
		{
			name: "openshift",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				groupVersions: []string{"apps.openshift.io/v1"},
			}),
			want: "openShift",
		},
		{
			name: "kurl",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"kurl.sh/cluster": "true",
							},
						},
					},
				},
			}),
			want: "kurl",
		},
		{
			name: "aks",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"kubernetes.azure.com/role": "agent",
							},
						},
					},
				},
			}),
			want: "aks",
		},
		{
			name: "eks",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						Spec: corev1.NodeSpec{
							ProviderID: "aws:providerid",
						},
					},
				},
			}),
			want: "eks",
		},
		{
			name: "gke",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						Spec: corev1.NodeSpec{
							ProviderID: "gce:providerid",
						},
					},
				},
			}),
			want: "gke",
		},
		{
			name: "ibm",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						Spec: corev1.NodeSpec{
							ProviderID: "ibm:providerid",
						},
					},
				},
			}),
			want: "ibm",
		},
		{
			name: "oke",
			clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
				objects: []runtime.Object{
					&corev1.Node{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"oci.oraclecloud.com/fault-domain": "domain-1",
							},
						},
					},
				},
			}),
			want: "oke",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := StaticCtx{
				clientset: tt.clientset,
			}

			if got := ctx.distribution(); got != tt.want {
				t.Errorf("StaticCtx.distribution() = %v, want %v", got, tt.want)
			}
		})
	}
}
