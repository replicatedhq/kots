package template

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestStaticContext_kubeSeal_badCert(t *testing.T) {
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
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
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
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
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
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
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
	req := require.New(t)

	builder := Builder{}
	builder.AddCtx(StaticCtx{})
	validateAndClearCaCert(req, builder)
}

func TestTlsCertFromCa(t *testing.T) {
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
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
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
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
