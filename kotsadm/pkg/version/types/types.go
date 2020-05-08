package types

type AppVersion struct {
	Sequence     int64
	UpdateCursor int
	VersionLabel string
}
