package types

import (
	"testing"

	"github.com/blang/semver"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/stretchr/testify/require"
)

func Test_SortDownstreamVersions(t *testing.T) {
	tests := []struct {
		name     string
		bySemver bool
		versions []*DownstreamVersion
		want     []*DownstreamVersion
	}{
		{
			name:     "mixed labels",
			bySemver: true,
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
			name:     "semver only",
			bySemver: true,
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
			name:     "none semver only",
			bySemver: true,
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
			name:     "one item",
			bySemver: true,
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
			bySemver: true,
			versions: []*DownstreamVersion{},
			want:     []*DownstreamVersion{},
		},
		{
			name:     "one more",
			bySemver: true,
			versions: []*DownstreamVersion{
				{
					VersionLabel: "1.2.5",
					Semver:       semverMustParseForTest("1.2.5"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.1.4",
					Semver:       semverMustParseForTest("1.1.4"),
					Sequence:     3,
				},
				{
					VersionLabel: "2.1.3",
					Semver:       semverMustParseForTest("2.1.3"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.1.2",
					Semver:       semverMustParseForTest("1.1.2"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.1.1",
					Semver:       semverMustParseForTest("1.1.1"),
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "2.1.3",
					Semver:       semverMustParseForTest("2.1.3"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.2.5",
					Semver:       semverMustParseForTest("1.2.5"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.1.4",
					Semver:       semverMustParseForTest("1.1.4"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.1.2",
					Semver:       semverMustParseForTest("1.1.2"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.1.1",
					Semver:       semverMustParseForTest("1.1.1"),
					Sequence:     0,
				},
			},
		},
		{
			name:     "sort by sequence",
			bySemver: false,
			versions: []*DownstreamVersion{
				{
					VersionLabel: "1.1.1",
					Semver:       semverMustParseForTest("1.1.1"),
					Sequence:     0,
				},
				{
					VersionLabel: "1.2.5",
					Semver:       semverMustParseForTest("1.2.5"),
					Sequence:     4,
				},
				{
					VersionLabel: "2.1.3",
					Semver:       semverMustParseForTest("2.1.3"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.1.4",
					Semver:       semverMustParseForTest("1.1.4"),
					Sequence:     3,
				},
				{
					VersionLabel: "1.1.2",
					Semver:       semverMustParseForTest("1.1.2"),
					Sequence:     1,
				},
			},
			want: []*DownstreamVersion{
				{
					VersionLabel: "1.2.5",
					Semver:       semverMustParseForTest("1.2.5"),
					Sequence:     4,
				},
				{
					VersionLabel: "1.1.4",
					Semver:       semverMustParseForTest("1.1.4"),
					Sequence:     3,
				},
				{
					VersionLabel: "2.1.3",
					Semver:       semverMustParseForTest("2.1.3"),
					Sequence:     2,
				},
				{
					VersionLabel: "1.1.2",
					Semver:       semverMustParseForTest("1.1.2"),
					Sequence:     1,
				},
				{
					VersionLabel: "1.1.1",
					Semver:       semverMustParseForTest("1.1.1"),
					Sequence:     0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			versions := &DownstreamVersions{
				AllVersions: tt.versions,
			}
			SortDownstreamVersions(versions, tt.bySemver)

			req.Equal(tt.want, versions.AllVersions)
		})
	}
}

func semverMustParseForTest(str string) *semver.Version {
	v := semver.MustParse(str)
	return &v
}

func Test_SortDownstreamVersionsByCursor(t *testing.T) {
	tests := []struct {
		name     string
		versions []*DownstreamVersion
		want     []*DownstreamVersion
	}{
		{
			name: "basic - no change",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
			},
		},
		{
			name: "basic - reverse",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
			},
		},
		{
			name: "random",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     1,
				},
			},
		},
		{
			name: "same cursor, different sequence",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     2,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     2,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
			},
		},
		{
			name: "higher cursor, lower sequence",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     3,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     2,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     3,
				},
			},
		},
		{
			name: "mixed",
			versions: []*DownstreamVersion{
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     5,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     4,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     2,
				},
				{
					UpdateCursor: "4",
					Cursor:       cursorMustParseForTest("4"),
					Sequence:     0,
				},
			},
			want: []*DownstreamVersion{
				{
					UpdateCursor: "4",
					Cursor:       cursorMustParseForTest("4"),
					Sequence:     0,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     3,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     2,
				},
				{
					UpdateCursor: "3",
					Cursor:       cursorMustParseForTest("3"),
					Sequence:     1,
				},
				{
					UpdateCursor: "2",
					Cursor:       cursorMustParseForTest("2"),
					Sequence:     4,
				},
				{
					UpdateCursor: "1",
					Cursor:       cursorMustParseForTest("1"),
					Sequence:     5,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			SortDownstreamVersionsByCursor(tt.versions)

			req.Equal(tt.want, tt.versions)
		})
	}
}

func cursorMustParseForTest(str string) *cursor.Cursor {
	c := cursor.MustParse(str)
	return &c
}
