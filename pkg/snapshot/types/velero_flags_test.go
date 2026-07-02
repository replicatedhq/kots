package types

import (
	"reflect"
	"testing"
)

func TestVeleroFSBackupFlags(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{"--use-node-agent"}},
		{"garbage", []string{"--use-node-agent"}},
		{"1.9.0", []string{"--use-restic"}},
		{"1.9.9", []string{"--use-restic"}},
		{"1.10.0", []string{"--use-node-agent", "--uploader-type=restic"}},
		{"1.16.2", []string{"--use-node-agent", "--uploader-type=restic"}},
		{"1.16.99", []string{"--use-node-agent", "--uploader-type=restic"}},
		{"1.17.0", []string{"--use-node-agent"}},
		{"v1.17.2", []string{"--use-node-agent"}},
		{"1.17.0-rc.1", []string{"--use-node-agent"}},
		{"2.0.0", []string{"--use-node-agent"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := VeleroFSBackupFlags(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("VeleroFSBackupFlags(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestVeleroSupportsLVP(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},      // unknown: don't block
		{"garbage", true}, // unparseable: don't block
		{"1.9.0", true},
		{"1.16.2", true},
		{"1.16.99", true},
		{"1.17.0", false},
		{"v1.17.0", false},
		{"1.17.0-rc.1", false},
		{"2.0.0", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := VeleroSupportsLVP(tc.in); got != tc.want {
				t.Errorf("VeleroSupportsLVP(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
