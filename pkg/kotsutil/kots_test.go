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
	analyzeMetaStrictFalseInt := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			StrVal: "0",
		},
	}
	analyzeMetaStrictTrueInt := troubleshootv1beta2.AnalyzeMeta{
		Strict: multitype.BoolOrString{
			StrVal: "1",
		},
	}
	tests := []struct {
		name      string
		preflight *troubleshootv1beta2.Preflight
		want      bool
		wantErr   bool
	}{
		{
			name:      "expect false when preflight is nil",
			preflight: nil,
			want:      false,
			wantErr:   false,
		}, {
			name: "expect false when preflight spec is empty",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{},
			},
			want:    false,
			wantErr: false,
		}, {
			name: "expect false when preflight spec's analyzers is nil",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: nil,
				},
			},
			want:    false,
			wantErr: false,
		}, {
			name: "expect false when preflight spec's analyzers is empty",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{},
				},
			},
			want:    false,
			wantErr: false,
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
			want:    false,
			wantErr: false,
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
			want:    false,
			wantErr: false,
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
			want:    false,
			wantErr: false,
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
			want:    false,
			wantErr: false,
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
			want:    true,
			wantErr: false,
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
			want:    true,
			wantErr: false,
		}, {
			name: "expect true when preflight spec's analyzer has anlyzer with strict int: 1",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictTrueInt},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		}, {
			name: "expect false when preflight spec's analyzer has anlyzer with strict int: 0",
			preflight: &troubleshootv1beta2.Preflight{
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{AnalyzeMeta: analyzeMetaStrictFalseInt},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasStrictPreflights(tt.preflight)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasStrictPreflights error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
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
