package search

import (
	"embed"
	"net/http"
)

//go:embed openapi.json docs.html
var docsFS embed.FS

// HandleOpenAPI serves the OpenAPI JSON.
func (h *Handler) HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := docsFS.ReadFile("openapi.json")
	if err != nil {
		http.Error(w, "openapi not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// HandleDocs serves Swagger UI.
func (h *Handler) HandleDocs(w http.ResponseWriter, r *http.Request) {
	data, err := docsFS.ReadFile("docs.html")
	if err != nil {
		http.Error(w, "docs not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
