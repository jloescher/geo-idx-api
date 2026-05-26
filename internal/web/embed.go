package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

// StaticFS returns the embedded static asset filesystem (css/, js/).
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFiles, "static")
}
