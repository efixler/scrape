package admin

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed htdocs/assets/*
var assetsDir embed.FS

func assetsHandler() http.Handler {
	d, err := fs.Sub(assetsDir, "htdocs/assets")
	if err != nil {
		panic(err)
	}
	// TODO: Disable directory listing
	return http.StripPrefix("/assets", http.FileServer(http.FS(d)))
}
