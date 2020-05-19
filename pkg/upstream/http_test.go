package upstream

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_privateUpstreamGetRequest(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		unsignedLicense *kotsv1beta1.UnsignedLicense
		cursor          ReplicatedCursor
	}{
		{
			name:   "basic get",
			method: "GET",
			unsignedLicense: &kotsv1beta1.UnsignedLicense{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "UnsignedLicense",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "basic-get",
				},
				Spec: kotsv1beta1.UnsignedLicenseSpec{
					Endpoint: "http://test",
					Slug:     "slug",
				},
			},
			cursor: ReplicatedCursor{
				Cursor: "c",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			req := require.New(t)

			p := PrivateUpstream{}
			actual, err := p.getRequest(test.method, test.unsignedLicense, test.cursor)
			req.NoError(err)

			headers := actual.Header
			assert.NotEmpty(t, headers.Get("User-Agent"))
			assert.NotEmpty(t, headers.Get("Authorization"))

			expectedAuthHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", test.unsignedLicense.Name, test.unsignedLicense.Name)))
			assert.Equal(t, fmt.Sprintf("Basic %s", expectedAuthHeader), headers.Get("Authorization"))

			assert.Equal(t, actual.Method, strings.ToUpper(test.method))

			assert.Equal(t, actual.URL.RequestURI(), fmt.Sprintf("/release/%s?channelSequence=%s", test.unsignedLicense.Spec.Slug, test.cursor.Cursor))
		})
	}
}
