package web

import (
	"embed"
)

// Content holds our (already built) web content as embedded files in the binary
//go:embed dist/*
var Content embed.FS
