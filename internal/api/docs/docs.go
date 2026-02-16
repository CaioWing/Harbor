package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html openapi.yaml
var assets embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(assets, ".")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "swagger assets not available", http.StatusInternalServerError)
		})
	}

	return http.FileServer(http.FS(sub))
}
