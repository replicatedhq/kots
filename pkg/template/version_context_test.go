package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionContext(t *testing.T) {
	req := require.New(t)

	// a properly populated versionCtx - should return the appropriate values
	ctx := versionCtx{
		info: &VersionInfo{
			Sequence:     5,
			Cursor:       "five",
			ChannelName:  "chanFive",
			VersionLabel: "verFive",
			IsRequired:   true,
			ReleaseNotes: "this is five",
			IsAirgap:     true,
		},
	}

	// an unpopulated versionCtx - should not error/panic
	nilCtx := versionCtx{}

	req.Equal(int64(5), ctx.sequence())
	req.Equal(int64(-1), nilCtx.sequence())

	req.Equal("five", ctx.cursor())
	req.Equal("", nilCtx.cursor())

	req.Equal("chanFive", ctx.channelName())
	req.Equal("", nilCtx.channelName())

	req.Equal("verFive", ctx.versionLabel())
	req.Equal("", nilCtx.versionLabel())

	req.Equal(true, ctx.isRequired())
	req.Equal(false, nilCtx.isRequired())

	req.Equal("this is five", ctx.releaseNotes())
	req.Equal("", nilCtx.releaseNotes())

	req.Equal(true, ctx.isAirgap())
	req.Equal(false, nilCtx.isAirgap())
}
