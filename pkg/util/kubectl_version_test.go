package util

import (
	"testing"
)

func Test_matchKnownVersion(t *testing.T) {
	tests := []struct {
		name       string
		userString string
		want       string
	}{
		{
			name:       "1",
			userString: "1",
			want:       "",
		},
		{
			name:       "notexist",
			userString: "1.11.5",
			want:       "",
		},
		{
			name:       "exact",
			userString: "1.16.15",
			want:       "v1.16",
		},
		{
			name:       "wrong patch",
			userString: "1.16.3",
			want:       "v1.16",
		},
		{
			name:       "1.14.x",
			userString: "1.14.x",
			want:       "v1.14",
		},
		{
			name:       "<1.15.0",
			userString: "<1.15.0",
			want:       "v1.14",
		},
		{
			name:       ">1.15.0 <1.17.0",
			userString: ">1.15.0 <1.17.0",
			want:       "v1.16",
		},
		{
			name:       "<1.17.0",
			userString: "<1.17.0",
			want:       "v1.16",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchKnownVersion(tt.userString); got != tt.want {
				t.Errorf("matchKnownVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
