package appstate

import (
	"testing"
)

func Test_getResourceKindCommonName(t *testing.T) {
	type args struct {
		a string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "deployment",
			args: args{a: "deploy"},
			want: "deployment",
		},
		{
			name: "statefulset",
			args: args{a: "sts"},
			want: "statefulset",
		},
		{
			name: "daemonset",
			args: args{a: "ds"},
			want: "daemonset",
		},
		{
			name: "uppercase",
			args: args{a: "StatefulSet"},
			want: "statefulset",
		},
		{
			name: "unknown",
			args: args{a: "blah"},
			want: "blah",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getResourceKindCommonName(tt.args.a); got != tt.want {
				t.Errorf("getResourceKindCommonName() = %v, want %v", got, tt.want)
			}
		})
	}
}
