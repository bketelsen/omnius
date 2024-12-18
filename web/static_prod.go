//go:build !dev
// +build !dev

package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	hashFS "github.com/benbjohnson/hashfs"
)

//go:embed static/*
var staticFS embed.FS
var staticRootFS, _ = fs.Sub(staticFS, "static")

func static(logger *slog.Logger) http.Handler {
	logger.Info("Static assets are embedded")
	return hashFS.FileServer(staticRootFS)
}
