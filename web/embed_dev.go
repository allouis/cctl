//go:build !embed

package web

import (
	"io/fs"
	"os"
)

// App returns an empty filesystem in dev mode. The frontend is served
// by Vite's dev server, not the Go binary.
func App() fs.FS {
	return os.DirFS("web/app/dist")
}
