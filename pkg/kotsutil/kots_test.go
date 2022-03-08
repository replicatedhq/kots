package kotsutil

import (
	"testing"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

func TestKotsKinds_HasStrictPreflights(t *testing.T) {

	analyzeMetaStrictFalseStr := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			StrVal: "false",
		},
	}
	analyzeMetaStrictInvalidStr := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			StrVal: "invalid",
		},
	}
	analyzeMetaStrictTrueStr := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			StrVal: "true",
		},
	}
	analyzeMetaStrictFalseBool := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			Type:    multitype.Bool,
			BoolVal: false,
		},
	}
	analyzeMetaStrictTrueBool := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			Type:    multitype.Bool,
			BoolVal: true,
		},
	}
	tests := []struct {
		name      string
		preflight *troubleshootv1beta2.Preflight
		want      bool
	}{
		{
			name:      "expect false when preflight is nil",
			preflight: nil,
			want:      false,
		}, {
			name: "expect false when preflight spec is empty",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzers is nil",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: nil,
				},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzers is empty",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{},
				},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzser has nil anlyzer",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: nil,
						},
					},
				},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzser has anlyzer with strict str: false",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictFalseStr},
						},
					},
				},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzser has anlyzer with strict bool: false",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictFalseBool},
						},
					},
				},
			},
			want: false,
		}, {
			name: "expect false when preflight spec's analyzser has anlyzer with strict str: invalid",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictInvalidStr},
						},
					},
				},
			},
			want: false,
		}, {
			name: "expect true when preflight spec's analyzser has anlyzer with strict str: true",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictTrueStr},
						},
					},
				},
			},
			want: true,
		}, {
			name: "expect true when preflight spec's analyzer has anlyzer with strict bool: true",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictTrueBool},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasStrictPreflights(tt.preflight); got != tt.want {
				t.Errorf("HasStrictPreflights() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStrictPreflightFailing(t *testing.T) {
	tests := []struct {
		name            string
		preflightResult *troubleshootpreflight.UploadPreflightResults
		want            bool
	}{
		{
			name:            "expect true when preflightResult is nil",
			preflightResult: nil,
			want:            true,
		}, {
			name: "expect true when preflightResult.Results is nil",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: nil,
			},
			want: true,
		}, {
			name: "expect true when preflightResult.Results is empty",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{},
			},
			want: true,
		}, {
			name: "expect false when preflightResult.Results has result with strict false, IsFail false",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{Strict: false, IsFail: false},
				},
			},
			want: false,
		}, {
			name: "expect false when preflightResult.Results has result with strict false, IsFail true",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{Strict: false, IsFail: true},
				},
			},
			want: false,
		}, {
			name: "expect true when preflightResult.Results has result with strict true, IsFail true",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{Strict: true, IsFail: true},
				},
			},
			want: true,
		}, {
			name: "expect true when preflightResult.Results has multiple results where atleast result has strict true, IsFail true",
			preflightResult: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{Strict: true, IsFail: true},
					{Strict: false, IsFail: true},
					{Strict: true, IsFail: false},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrictPreflightFailing(tt.preflightResult); got != tt.want {
				t.Errorf("IsStrictPreflightFailing() = %v, want %v", got, tt.want)
			}
		})
	}
}
