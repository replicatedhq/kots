package client

import (
	"testing"
)

func Test_shouldWaitForResourceDeletion(t *testing.T) {
	type args struct {
		kind     string
		waitFlag bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expect true when wait flag is true",
			args: args{
				kind:     "Pod",
				waitFlag: true,
			},
			want: true,
		}, {
			name: "expect false when wait flag is false",
			args: args{
				kind:     "Pod",
				waitFlag: false,
			},
			want: false,
		}, {
			name: "expect false when kind is PersistentVolumeClaim",
			args: args{
				kind:     "PersistentVolumeClaim",
				waitFlag: true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWaitForResourceDeletion(tt.args.kind, tt.args.waitFlag); got != tt.want {
				t.Errorf("shouldWaitForResourceDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}
