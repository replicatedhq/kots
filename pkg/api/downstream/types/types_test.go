package types

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
)

func Test_SortDownstreamVersions(t *testing.T) {
	tests := []struct {
		name     string
		versions []*DownstreamVersion
		want     []*DownstreamVersion
	}{
		{
			name: "mixed labels",
			versions: []*DownstreamVersion{
				{
					VersionLabel: "notsemver2",
					Semver:       nil,
					Sequence:     13,
				},
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     12,
				},
				{
					VersionLabel: "1.0.284",
					Semver:       semverMustParseForTest("1.0.284"),
					Sequence:     11,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     10,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     9,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     8,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     7,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     6,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     5,
				},
				{
					VersionLabel: "1.0.283",
					Semver:       semverMustParseForTest("1.0.283"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.0.282",
					Semver:       semverMustParseForTest("1.0.282"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.0.281",
					Semver:       semverMustParseForTest("1.0.281"),
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "notsemver2",
					Semver:       nil,
					Sequence:     13,
				},
				{
					VersionLabel: "1.0.284",
					Semver:       semverMustParseForTest("1.0.284"),
					Sequence:     11,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     10,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     9,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     8,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     7,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     6,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     5,
				},
				{
					VersionLabel: "1.0.283",
					Semver:       semverMustParseForTest("1.0.283"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.0.282",
					Semver:       semverMustParseForTest("1.0.282"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.0.281",
					Semver:       semverMustParseForTest("1.0.281"),
					Sequence:     0,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     12,
				},
			},
		},
		{
			name: "semver only",
			versions: []*DownstreamVersion{
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     6,
				},
				{
					VersionLabel: "1.0.284",
					Semver:       semverMustParseForTest("1.0.284"),
					Sequence:     5,
				},
				{
					VersionLabel: "1.0.283",
					Semver:       semverMustParseForTest("1.0.283"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.0.282",
					Semver:       semverMustParseForTest("1.0.282"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.0.281",
					Semver:       semverMustParseForTest("1.0.281"),
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "1.0.284",
					Semver:       semverMustParseForTest("1.0.284"),
					Sequence:     5,
				},
				{
					VersionLabel: "1.0.283",
					Semver:       semverMustParseForTest("1.0.283"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.0.282",
					Semver:       semverMustParseForTest("1.0.282"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.0.281",
					Semver:       semverMustParseForTest("1.0.281"),
					Sequence:     0,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.0.279",
					Semver:       semverMustParseForTest("1.0.279"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     6,
				},
			},
		},
		{
			name: "none semver only",
			versions: []*DownstreamVersion{
				{
					VersionLabel: "notsemver2",
					Semver:       nil,
					Sequence:     6,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     5,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     4,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     3,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     2,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     1,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "notsemver2",
					Semver:       nil,
					Sequence:     6,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     5,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     4,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     3,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     2,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     1,
				},
				{
					VersionLabel: "notasemver1",
					Semver:       nil,
					Sequence:     0,
				},
			},
		},
		{
			name: "one item",
			versions: []*DownstreamVersion{
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "1.0.185",
					Semver:       semverMustParseForTest("1.0.185"),
					Sequence:     0,
				},
			},
		},
		{
			name:     "empty",
			versions: []*DownstreamVersion{},
			want:     []*DownstreamVersion{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			versions := &DownstreamVersions{
				AllVersions: tt.versions,
			}
			SortDownstreamVersions(versions)

			req.Equal(tt.want, versions.AllVersions)
		})
	}
}

func semverMustParseForTest(str string) *semver.Version {
	v := semver.MustParse(str)
	return &v
}
