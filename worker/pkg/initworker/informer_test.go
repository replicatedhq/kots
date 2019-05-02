package initworker

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/state"
)

func Test_createWatchName(t *testing.T) {
	tests := []struct {
		name     string
		metadata state.Metadata
		uri      string
		want     string
	}{
		{
			name:     "metadataName present",
			metadata: state.Metadata{Name: "abc123"},
			uri:      "not relevant",
			want:     "abc123",
		},
		{
			name:     "app slug present",
			metadata: state.Metadata{AppSlug: "slug123"},
			uri:      "not relevant",
			want:     "slug123",
		},
		{
			name:     "app slug ship app, no slug present in state",
			metadata: state.Metadata{},
			uri:      "staging.replicated.app/ec2-metadata-get?license_id=random-characters-go-here",
			want:     "staging.replicated.app/ec2-metadata-get?license_id=random-characters-go-here",
		},
		{
			name:     "metadataName not present, unparsable uri",
			metadata: state.Metadata{},
			uri:      "not parsable",
			want:     "not parsable",
		},
		{
			name:     "github uri",
			metadata: state.Metadata{},
			uri:      "https://github.com/jenkinsci/kubernetes-operator/tree/v0.0.4/deploy",
			want:     "jenkinsci/kubernetes-operator@v0.0.4",
		},
		{
			name:     "githubusercontent uri",
			metadata: state.Metadata{},
			uri:      "https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/master/jaeger-production-template.yml",
			want:     "jaegertracing/jaeger-kubernetes@master",
		},
		{
			name:     "no version in uri",
			metadata: state.Metadata{},
			uri:      "https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/",
			want:     "jaegertracing/jaeger-kubernetes",
		},
		{
			name:     "no trailing slash or version in uri",
			metadata: state.Metadata{},
			uri:      "https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes",
			want:     "jaegertracing/jaeger-kubernetes",
		},
		{
			name:     "weave.works URL",
			metadata: state.Metadata{},
			uri:      "https://cloud.weave.works/k8s/n-e_t.yaml?k8s-version=1.11.5\u0026omit-support-info=true",
			want:     "cloud.weave.works/k8s/n-e_t.yaml",
		},
		{
			name:     "stuttering github repo URL",
			metadata: state.Metadata{},
			uri:      "https://github.com/cloudflare/cloudflare-ingress-controller/blob/master/deploy/argo-tunnel.yaml",
			want:     "cloudflare-ingress-controller@master",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createWatchName(tt.metadata, tt.uri); got != tt.want {
				t.Errorf("createWatchName() = %v, want %v", got, tt.want)
			}
		})
	}
}
