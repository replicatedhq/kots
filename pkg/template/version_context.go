package template

import (
	"text/template"
)

type VersionInfo struct {
	Sequence     int64  // the installation sequence. Always 0 when being freshly installed, etc
	Cursor       string // the upstream version cursor - integers for kots apps, may be semvers for helm charts
	ChannelName  string // the name of the channel that the current version was from (kots apps only)
	VersionLabel string // a pretty version label if provided (kots apps only)
	ReleaseNotes string // the release notes for the given version (kots apps only)
	IsAirgap     bool   // is this an airgap app (kots apps only)
}

type versionCtx struct {
	info *VersionInfo
}

func newVersionCtx(info *VersionInfo) versionCtx {
	return versionCtx{info: info}
}

// FuncMap represents the available functions in the versionCtx.
func (ctx versionCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"Sequence":     ctx.sequence,
		"Cursor":       ctx.cursor,
		"ChannelName":  ctx.channelName,
		"VersionLabel": ctx.versionLabel,
		"ReleaseNotes": ctx.releaseNotes,
		"IsAirgap":     ctx.isAirgap,
	}
}

func (ctx versionCtx) sequence() int64 {
	if ctx.info == nil {
		return -1
	}
	return ctx.info.Sequence
}

func (ctx versionCtx) cursor() string {
	if ctx.info == nil {
		return ""
	}
	return ctx.info.Cursor
}

func (ctx versionCtx) channelName() string {
	if ctx.info == nil {
		return ""
	}
	return ctx.info.ChannelName
}

func (ctx versionCtx) versionLabel() string {
	if ctx.info == nil {
		return ""
	}
	return ctx.info.VersionLabel
}

func (ctx versionCtx) releaseNotes() string {
	if ctx.info == nil {
		return ""
	}
	return ctx.info.ReleaseNotes
}

func (ctx versionCtx) isAirgap() bool {
	if ctx.info == nil {
		return false
	}
	return ctx.info.IsAirgap
}
