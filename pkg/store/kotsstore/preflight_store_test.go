package kotsstore

import (
	"encoding/json"
	"testing"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/rqlite/gorqlite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	strictTruePreflightSpec = &troubleshootv1beta2.Preflight{
		TypeMeta: v1.TypeMeta{
			Kind:       "Preflight",
			APIVersion: "troubleshoot.sh/v1beta2",
		},
		Spec: troubleshootv1beta2.PreflightSpec{
			Analyzers: []*troubleshootv1beta2.Analyze{
				{
					ClusterVersion: &troubleshootv1beta2.ClusterVersion{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
							Strict: &multitype.BoolOrString{StrVal: "true"},
						},
					},
				},
			},
		},
	}

	strictFalsePreflightSpec = &troubleshootv1beta2.Preflight{
		TypeMeta: v1.TypeMeta{
			Kind:       "Preflight",
			APIVersion: "troubleshoot.sh/v1beta2",
		},
		Spec: troubleshootv1beta2.PreflightSpec{
			Analyzers: []*troubleshootv1beta2.Analyze{
				{
					ClusterVersion: &troubleshootv1beta2.ClusterVersion{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
							Strict: &multitype.BoolOrString{StrVal: "false"},
						},
					},
				},
			},
		},
	}

	strictFailTruePreflightResultSpec = &troubleshootpreflight.UploadPreflightResults{
		Results: []*troubleshootpreflight.UploadPreflightResult{
			{
				Strict: true,
				IsFail: true,
			},
		},
	}

	strictFailFalsePreflightResultSpec = &troubleshootpreflight.UploadPreflightResults{
		Results: []*troubleshootpreflight.UploadPreflightResult{
			{
				Strict: true,
				IsFail: false,
			},
		},
	}
)

func toSqlString(t *testing.T, val interface{}) gorqlite.NullString {
	b, err := json.Marshal(val)
	if err != nil {
		t.Fatalf("hasFailingStrictPreflights() json Unmarshall error = %v", err)
	}
	return gorqlite.NullString{
		String: string(b),
		Valid:  true,
	}
}

func Test_hasFailingStrictPreflights(t *testing.T) {
	tests := []struct {
		name               string
		preflightSpecStr   gorqlite.NullString
		preflightResultStr gorqlite.NullString
		want               bool
		wantErr            bool
	}{
		{
			name:               "expect true, nil when preflightSpec with strict:true analyzer and result IsFail:true",
			preflightSpecStr:   toSqlString(t, strictTruePreflightSpec),
			preflightResultStr: toSqlString(t, strictFailTruePreflightResultSpec),
			want:               true,
			wantErr:            false,
		}, {
			name:               "expect false, nil when preflightSpec with strict:true analyzer and result IsFail:false",
			preflightSpecStr:   toSqlString(t, strictTruePreflightSpec),
			preflightResultStr: toSqlString(t, strictFailFalsePreflightResultSpec),
			want:               false,
			wantErr:            false,
		}, {
			name:               "expect false, nil when preflightSpec with strict:false analyzer and result IsFail:true",
			preflightSpecStr:   toSqlString(t, strictFalsePreflightSpec),
			preflightResultStr: toSqlString(t, strictFailTruePreflightResultSpec),
			want:               false,
			wantErr:            false,
		}, {
			name:               "expect false, nil when preflightSpec with strict:false analyzer and result IsFail:false",
			preflightSpecStr:   toSqlString(t, strictFalsePreflightSpec),
			preflightResultStr: toSqlString(t, strictFailFalsePreflightResultSpec),
			want:               false,
			wantErr:            false,
		}, {
			name:               "expect false, error when preflightSpec has a parse error",
			preflightSpecStr:   gorqlite.NullString{Valid: true, String: "invalid"},
			preflightResultStr: toSqlString(t, strictFailFalsePreflightResultSpec),
			want:               false,
			wantErr:            true,
		}, {
			name:               "expect false, error when preflightResultSpec has a parse error",
			preflightSpecStr:   toSqlString(t, strictTruePreflightSpec),
			preflightResultStr: gorqlite.NullString{Valid: true, String: "invalid"},
			want:               false,
			wantErr:            true,
		}, {
			name:               "expect false, error when preflightSpec with strict:true analyzer and preflightResultSpec has a empty string",
			preflightSpecStr:   toSqlString(t, strictTruePreflightSpec),
			preflightResultStr: gorqlite.NullString{Valid: true, String: ""},
			want:               false,
			wantErr:            false,
		}, {
			name:               "expect false, error when preflightSpec with strict:false analyzer and preflightResultSpec has a empty string",
			preflightSpecStr:   toSqlString(t, strictFalsePreflightSpec),
			preflightResultStr: gorqlite.NullString{Valid: true, String: ""},
			want:               false,
			wantErr:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &KOTSStore{}
			got, err := s.hasFailingStrictPreflights(tt.preflightSpecStr, tt.preflightResultStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("hasFailingStrictPreflights() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hasFailingStrictPreflights() = %v, want %v", got, tt.want)
			}
		})
	}
}
