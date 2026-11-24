package httpapi

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	boxd "github.com/xuthus5/boxd"
)

func (s *Server) staticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, r, newAPIError(http.StatusNotFound, "not_found", "http.router", "API endpoint was not found", ""))
			return
		}
		if r.URL.Path == "/favicon.ico" {
			http.Redirect(w, r, "/favicon.svg", http.StatusTemporaryRedirect)
			return
		}
		requestPath := path.Clean("/" + r.URL.Path)
		if requestPath == "/" {
			requestPath = "/index.html"
		}
		embeddedPath := boxd.WebRoot + requestPath
		body, err := fs.ReadFile(boxd.WebFiles, embeddedPath)
		if err != nil {
			if strings.HasPrefix(requestPath, "/assets/") {
				http.NotFound(w, r)
				return
			}
			body, err = fs.ReadFile(boxd.WebFiles, boxd.WebRoot+"/index.html")
			if err != nil {
				http.Error(w, "web assets are unavailable", http.StatusInternalServerError)
				return
			}
			requestPath = "/index.html"
		}
		contentType := mime.TypeByExtension(path.Ext(requestPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if (strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "javascript")) && !strings.Contains(strings.ToLower(contentType), "charset=") {
			contentType += "; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		if requestPath == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		http.ServeContent(w, r, requestPath, s.startedAt.Round(time.Second), strings.NewReader(string(body)))
	})
}
