package embeddedcluster

import (
	"reflect"
	"testing"
)

func Test_maskLicenseIDInArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
		{
			name: "no license id args",
			args: []string{"upgrade", "--app-slug", "example", "--channel-slug", "stable"},
			want: []string{"upgrade", "--app-slug", "example", "--channel-slug", "stable"},
		},
		{
			name: "license id with equals format",
			args: []string{"upgrade", "--license-id=abc123", "--app-slug", "example"},
			want: []string{"upgrade", "--license-id=REDACTED", "--app-slug", "example"},
		},
		{
			name: "license id with space format",
			args: []string{"upgrade", "--license-id", "abc123", "--app-slug", "example"},
			want: []string{"upgrade", "--license-id", "REDACTED", "--app-slug", "example"},
		},
		{
			name: "license id as last argument with space format",
			args: []string{"upgrade", "--app-slug", "example", "--license-id", "abc123"},
			want: []string{"upgrade", "--app-slug", "example", "--license-id", "REDACTED"},
		},
		{
			name: "multiple license id args",
			args: []string{"upgrade", "--license-id=abc123", "--app-slug", "example", "--license-id", "def456"},
			want: []string{"upgrade", "--license-id=REDACTED", "--app-slug", "example", "--license-id", "REDACTED"},
		},
		{
			name: "license id as last argument without value",
			args: []string{"upgrade", "--app-slug", "example", "--license-id"},
			want: []string{"upgrade", "--app-slug", "example", "--license-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskLicenseIDInArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("maskLicenseIDInArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
