package cli

import (
	"testing"

	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_decryptGroups(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-encryption",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"encryptionKey": []byte("IvWItkB8+ezMisPjSMBknT1PdKjBx7Xc/txZqOP8Y2Oe7+Jy"),
		},
	}
	clientset := testclient.NewSimpleClientset(secret)

	tests := []struct {
		name    string
		groups  []v1beta1.ConfigGroup
		want    []v1beta1.ConfigGroup
		wantErr error
	}{
		{
			name:    "no groups",
			groups:  []v1beta1.ConfigGroup{},
			want:    []v1beta1.ConfigGroup{},
			wantErr: nil,
		},
		{
			name: "item type password, and decrypt is true",
			groups: []v1beta1.ConfigGroup{
				{
					Items: []v1beta1.ConfigItem{
						{
							Type: "password",
							Value: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "DCvAuKERTaoFKQVcV3HBWw4=",
							},
						},
					},
				},
			},
			want: []v1beta1.ConfigGroup{
				{
					Items: []v1beta1.ConfigItem{
						{
							Type: "password",
							Value: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "d",
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "item type password, value missing but default set, and decrypt is true",
			groups: []v1beta1.ConfigGroup{
				{
					Items: []v1beta1.ConfigItem{
						{
							Type: "password",
							Default: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "DCvAuKERTaoFKQVcV3HBWw4=",
							},
						},
					},
				},
			},
			want: []v1beta1.ConfigGroup{
				{
					Items: []v1beta1.ConfigItem{
						{
							Type: "password",
							Default: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "d",
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decryptGroups(clientset, "namespace", tt.groups)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
