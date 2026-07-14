// Package ui serves the embedded report dashboard.
package ui

import (
	"embed"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

//go:embed static/index.html
var static embed.FS

var runFilePattern = regexp.MustCompile(`^run-[0-9]{8}-[0-9]{6}\.json$`)

type runSummary struct {
	File        string         `json:"file"`
	GeneratedAt string         `json:"generated_at"`
	Corpus      string         `json:"corpus"`
	OldBaseURL  string         `json:"old_base_url"`
	NewBaseURL  string         `json:"new_base_url"`
	Summary     map[string]int `json:"summary"`
	Total       int            `json:"total"`
}

func Serve(addr, reportDir string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		page, _ := static.ReadFile("static/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})

	mux.HandleFunc("/api/runs", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(reportDir)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		runs := []runSummary{}
		for _, e := range entries {
			if e.IsDir() || !runFilePattern.MatchString(e.Name()) {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(reportDir, e.Name()))
			if err != nil {
				continue
			}
			var meta struct {
				GeneratedAt string         `json:"generated_at"`
				Corpus      string         `json:"corpus"`
				OldBaseURL  string         `json:"old_base_url"`
				NewBaseURL  string         `json:"new_base_url"`
				Summary     map[string]int `json:"summary"`
				Results     []struct{}     `json:"results"`
			}
			if json.Unmarshal(raw, &meta) != nil {
				continue
			}
			runs = append(runs, runSummary{
				File:        e.Name(),
				GeneratedAt: meta.GeneratedAt,
				Corpus:      meta.Corpus,
				OldBaseURL:  meta.OldBaseURL,
				NewBaseURL:  meta.NewBaseURL,
				Summary:     meta.Summary,
				Total:       len(meta.Results),
			})
		}
		sort.Slice(runs, func(i, j int) bool { return runs[i].File > runs[j].File })
		writeJSON(w, runs)
	})

	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("file")
		if !runFilePattern.MatchString(name) {
			jsonError(w, http.StatusBadRequest, "invalid file name")
			return
		}
		raw, err := os.ReadFile(filepath.Join(reportDir, name))
		if err != nil {
			jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(raw)
	})

	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
