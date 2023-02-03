package main

import (
	"fmt"
	"testing"
)

const (
	TEST_PRIVATE_KEY = `Bag Attributes
localKeyID: 75 15 4A FC 72 DB A7 02 AD A2 2B 56 9F 9A F9 A5 A5 0D 5F F3
Key Attributes: <No Attributes>
-----BEGIN ENCRYPTED PRIVATE KEY-----
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
zyk=
-----END ENCRYPTED PRIVATE KEY-----
`
	TEST_CERTIFICATE = `Bag Attributes
    localKeyID: 75 15 4A FC 72 DB A7 02 AD A2 2B 56 9F 9A F9 A5 A5 0D 5F F3
subject=/CN=MyHost.com
issuer=/CN=MyHost.com
-----BEGIN CERTIFICATE-----
MIIEpjCCAo4CCQDBd+iafziSDjANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDDApN
eUhvc3QuY29tMB4XDTIyMTEwOTExMDE1NloXDTIzMTEwOTExMDE1NlowFTETMBEG
A1UEAwwKTXlIb3N0LmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AMSeME6lH2EkOz78MSbft6ejAGvbARjV3E5brYRL2UANBAXeojMrsUdIu65S/yxE
scM9AfA2d7Zqw0biaPIzABEMPkbrTph4aLRXsj9CmM5AbHj7FmPJ6WyTM86a1dJT
yxOTriiwtNrN5owbjN0GrVf5fruKm3+kA6Mj0kStT2jKNysIBxvS+aY3U0QbXaqM
uKzdlTFHJVp4jCLhRdIROVg4dQRUkBNBG2CwQIG07y8aaXwWxF5/ooCifRDyeONW
ac3VA/N7536AinnnHCJf6EXrTG3tiuLWD0hI3aPT5DFdZjLF/6yn4z0uxI6Lx4VX
8E4mEUnTWXoUAp8j52Ut5s9sjsrNDodF5E5QmRpapNmAhLEqFQw4R4BiXqqwbELK
YAlFnSwA+gSLpIyaoJyJAKgSxGw8C4zNL0RnRQvrcUEExWAgpXym0GBmJNCkO+Vc
wXjWNIirqlWB/K0N7ahcQb50ClOflHhg1mDHnngGS1AAzfBtyz548VGAmIHyHcwq
exNgQXv1007nXzRsGihvCHqKji6V4PAVMlgbycduREoaEzkVRAPcxoSeIBBjcTfP
Ti6eMXtKigJF1A5+3ZB0I0APgWdpfOkl6EI7xL5rV+6Tf5O9E0L58L73pPNFHuzO
uVAzvTrMbm+PVUhnmEmyddzQEvY41b62dBl5cVPLzXlTAgMBAAEwDQYJKoZIhvcN
AQELBQADggIBAI3ETdAifi6kyNPn1I+J6TaFDj9iwoO8GIAHTwAZ41pR+ooa1aqn
55OCu3ehmI8I35l6tySSsGRnr/RCRxt9xs4UiIk1+ZE2zJTb0myBYZWXGcCIJPZ8
6cILOJ+JBYCofJTqgtQjv1KezIASfvQWsb2/GjYz70PfyONY1FvEes4zIBV5ezPl
pqpmGlVR+kDCxr29h7+HvE6/FLaJgwx1o1/nMQsveVG+fRj2Qv7vDiweDAXbpGpu
fg4sHZS2G9E6ZW4ArVpSS0SmLJ2NMf7hl+xXWBK7tUUgwg/Tpk2CLrPr++oaZcH5
H2GDp/bh9JeryvNS6UZfnJqI+tcgWo18PYljeQwsnWUX2wYINFF6aNdx2pp6WDN7
zkpx0PnCgcOgacyjqt1cqEXiWPUKF8vipxwCQc/uNjgfdxJftrKvRjK8hwSMp7mU
wGnnjIgKx/rgaF7cFdpRdMofWCKQclTxoQ4lBkFP0I7WkMNLq8BYFMwL/Aytsz/3
U29Wuhlm3+3r1bEKDVrlbB45NDF7DiiQoVj8am0YGVfVhjaZ9CWcUtXPdXTrLhKo
ISlS5SumPuf/WYY9WBB/W9TaroA4PVIlGlhvqFvbPAtm8sWn5Fuy1/Iyyw/gjPj5
MsD4DATZj4d6wXi45GIbOoBJKrceOQVfxBXOD0X6FFLysK2wlKfY6FNu
-----END CERTIFICATE-----
`
)

func Test_getFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		certData []byte
		want     string
		wantErr  bool
	}{
		{
			name:     "valid cert",
			certData: []byte(fmt.Sprintf("%s/n%s", TEST_CERTIFICATE, TEST_PRIVATE_KEY)),
			want:     "75:15:4A:FC:72:DB:A7:02:AD:A2:2B:56:9F:9A:F9:A5:A5:0D:5F:F3",
			wantErr:  false,
		},
		{
			name:     "valid cert private key listed first",
			certData: []byte(fmt.Sprintf("%s/n%s", TEST_PRIVATE_KEY, TEST_CERTIFICATE)),
			want:     "75:15:4A:FC:72:DB:A7:02:AD:A2:2B:56:9F:9A:F9:A5:A5:0D:5F:F3",
			wantErr:  false,
		},
		{
			name:     "invalid cert provided",
			certData: []byte("invalid cert"),
			want:     "",
			wantErr:  true,
		},
		{
			name:     "no certificate type provided",
			certData: []byte(TEST_PRIVATE_KEY),
			want:     "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFingerprint(tt.certData)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFingerprint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getFingerprint() = %v, want %v", got, tt.want)
			}
		})
	}
}
