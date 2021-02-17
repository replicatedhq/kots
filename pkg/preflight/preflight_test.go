package preflight

import (
	"testing"

	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

func Test_getPreflightState(t *testing.T) {
	tests := []struct {
		name             string
		preflightResults *troubleshootpreflight.UploadPreflightResults
		want             string
	}{
		{
			name: "pass",
			preflightResults: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{},
					{},
				},
			},
			want: "pass",
		},
		{
			name: "warn",
			preflightResults: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsWarn: true},
					{},
				},
			},
			want: "warn",
		},
		{
			name: "fail",
			preflightResults: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsFail: true},
					{},
				},
			},
			want: "fail",
		},
		{
			name: "error",
			preflightResults: &troubleshootpreflight.UploadPreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsWarn: true},
					{},
				},
				Errors: []*troubleshootpreflight.UploadPreflightError{
					{},
				},
			},
			want: "fail",
		},
		{
			name:             "empty",
			preflightResults: &troubleshootpreflight.UploadPreflightResults{},
			want:             "pass",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := getPreflightState(test.preflightResults); got != test.want {
				t.Errorf("getPreflightState() = %v, want %v", got, test.want)
			}
		})
	}
}
