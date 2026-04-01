//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed app/dist/*
var app embed.FS

func App() fs.FS {
	sub, err := fs.Sub(app, "app/dist")
	if err != nil {
		panic(err)
	}
	return sub
}
