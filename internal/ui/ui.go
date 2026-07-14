// Package ui serves the embedded dashboard and the control-plane API:
// run reports, traffic capture (recording proxy), replay orchestration,
// and config editing.
package ui

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/pajamasi726/mocking-box/internal/capture"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/replay"
	"github.com/pajamasi726/mocking-box/internal/report"
)

//go:embed static/index.html
var static embed.FS

var (
	runFilePattern    = regexp.MustCompile(`^run-[0-9]{8}-[0-9]{6}\.json$`)
	corpusNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// Server is the long-running control plane behind `mockingbox ui`.
type Server struct {
	configPath string

	mu       sync.Mutex
	recorder *capture.Recorder
	replayMu sync.Mutex
	replayST replayStatus
}

type replayStatus struct {
	Running    bool   `json:"running"`
	Corpus     string `json:"corpus,omitempty"`
	Done       int    `json:"done"`
	Total      int    `json:"total"`
	Current    string `json:"current,omitempty"`
	LastReport string `json:"last_report,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

func NewServer(configPath string) *Server { return &Server{configPath: configPath} }

func (s *Server) loadConfig() (*config.Config, error) { return config.Load(s.configPath) }

func Serve(addr, configPath string) error {
	s := NewServer(configPath)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		page, _ := static.ReadFile("static/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})

	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("GET /api/run", s.getRun)
	mux.HandleFunc("GET /api/config", s.getConfig)
	mux.HandleFunc("PUT /api/config", s.putConfig)
	mux.HandleFunc("GET /api/corpora", s.listCorpora)
	mux.HandleFunc("GET /api/corpus", s.getCorpus)
	mux.HandleFunc("GET /api/capture/status", s.captureStatus)
	mux.HandleFunc("POST /api/capture/start", s.captureStart)
	mux.HandleFunc("POST /api/capture/stop", s.captureStop)
	mux.HandleFunc("GET /api/replay/status", s.replayStatusHandler)
	mux.HandleFunc("POST /api/replay/start", s.replayStart)

	return http.ListenAndServe(addr, mux)
}

// -- runs ---------------------------------------------------------------------

func (s *Server) reportDir() string {
	if cfg, err := s.loadConfig(); err == nil {
		return cfg.Report.Dir
	}
	return "./report"
}

func (s *Server) corpusDir() string {
	if cfg, err := s.loadConfig(); err == nil {
		return cfg.Corpus.Dir
	}
	return "./corpus"
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	dir := s.reportDir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runs := []map[string]any{}
	for _, e := range entries {
		if e.IsDir() || !runFilePattern.MatchString(e.Name()) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
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
		runs = append(runs, map[string]any{
			"file": e.Name(), "generated_at": meta.GeneratedAt, "corpus": meta.Corpus,
			"old_base_url": meta.OldBaseURL, "new_base_url": meta.NewBaseURL,
			"summary": meta.Summary, "total": len(meta.Results),
		})
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i]["file"].(string) > runs[j]["file"].(string) })
	writeJSON(w, runs)
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("file")
	if !runFilePattern.MatchString(name) {
		jsonError(w, http.StatusBadRequest, "invalid file name")
		return
	}
	raw, err := os.ReadFile(filepath.Join(s.reportDir(), name))
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw)
}

// -- config -------------------------------------------------------------------

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, configToJSON(cfg))
}

func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var incoming map[string]any
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	// round-trip through YAML into the typed config for validation
	yamlBytes, err := yaml.Marshal(incoming)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	tmp := s.configPath + ".tmp"
	if err := os.WriteFile(tmp, yamlBytes, 0o644); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := config.Load(tmp); err != nil {
		os.Remove(tmp)
		jsonError(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}
	if err := os.Rename(tmp, s.configPath); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg, _ := s.loadConfig()
	writeJSON(w, configToJSON(cfg))
}

// configToJSON renders the config in the same shape the YAML file uses.
func configToJSON(cfg *config.Config) map[string]any {
	stack := func(st config.Stack) map[string]any {
		m := map[string]any{"base_url": st.BaseURL}
		if st.MySQL != nil {
			m["mysql"] = map[string]any{
				"host": st.MySQL.Host, "port": st.MySQL.Port,
				"user": st.MySQL.User, "password": st.MySQL.Password,
			}
		}
		return m
	}
	sortRules := []map[string]string{}
	for _, sr := range cfg.Compare.SortArrays {
		sortRules = append(sortRules, map[string]string{"path": sr.Path, "by": sr.By})
	}
	return map[string]any{
		"old": stack(cfg.Old), "new": stack(cfg.New),
		"attribution": map[string]any{
			"quiet_ms": cfg.Attribution.QuietMs, "timeout_ms": cfg.Attribution.TimeoutMs,
			"check_innodb_trx": cfg.Attribution.TrxCheck(),
		},
		"noise": map[string]any{
			"response_paths": orEmpty(cfg.Noise.ResponsePaths),
			"columns":        orEmpty(cfg.Noise.Columns),
			"tables_ignore":  orEmpty(cfg.Noise.TablesIgnore),
		},
		"compare":         map[string]any{"sort_arrays": sortRules},
		"http_timeout_s":  cfg.HTTPTimeoutS,
		"compare_headers": orEmpty(cfg.CompareHeaders),
		"report":          map[string]any{"dir": cfg.Report.Dir},
		"corpus":          map[string]any{"dir": cfg.Corpus.Dir},
	}
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// -- corpora ------------------------------------------------------------------

func (s *Server) listCorpora(w http.ResponseWriter, r *http.Request) {
	dir := s.corpusDir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []map[string]any{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jsonl" && ext != ".har" {
			continue
		}
		info, _ := e.Info()
		count := 0
		if specs, err := corpus.Load(filepath.Join(dir, e.Name())); err == nil {
			count = len(specs)
		}
		entry := map[string]any{"name": e.Name(), "count": count}
		if info != nil {
			entry["size"] = info.Size()
			entry["modified_at"] = info.ModTime().Format("2006-01-02 15:04:05")
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
	writeJSON(w, out)
}

func (s *Server) getCorpus(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if !corpusNamePattern.MatchString(name) {
		jsonError(w, http.StatusBadRequest, "invalid corpus name")
		return
	}
	specs, err := corpus.Load(filepath.Join(s.corpusDir(), name))
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, specs)
}

// -- capture ------------------------------------------------------------------

func (s *Server) captureStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recorder == nil {
		writeJSON(w, capture.Status{Running: false})
		return
	}
	writeJSON(w, s.recorder.Status())
}

func (s *Server) captureStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Listen   string `json:"listen"`
		Upstream string `json:"upstream"`
		Corpus   string `json:"corpus"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Listen == "" || req.Upstream == "" || req.Corpus == "" {
		jsonError(w, http.StatusBadRequest, "listen, upstream, corpus are required")
		return
	}
	if !strings.HasSuffix(req.Corpus, ".jsonl") {
		req.Corpus += ".jsonl"
	}
	if !corpusNamePattern.MatchString(req.Corpus) {
		jsonError(w, http.StatusBadRequest, "invalid corpus name")
		return
	}
	if !strings.Contains(req.Listen, ":") {
		req.Listen = ":" + req.Listen
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recorder != nil && s.recorder.Status().Running {
		jsonError(w, http.StatusConflict, "capture already running")
		return
	}
	dir := s.corpusDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rec, err := capture.Start(req.Listen, req.Upstream, filepath.Join(dir, req.Corpus))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.recorder = rec
	writeJSON(w, rec.Status())
}

func (s *Server) captureStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recorder == nil {
		jsonError(w, http.StatusConflict, "no capture running")
		return
	}
	s.recorder.Stop()
	st := s.recorder.Status()
	s.recorder = nil
	writeJSON(w, st)
}

// -- replay -------------------------------------------------------------------

func (s *Server) replayStatusHandler(w http.ResponseWriter, r *http.Request) {
	s.replayMu.Lock()
	defer s.replayMu.Unlock()
	writeJSON(w, s.replayST)
}

func (s *Server) replayStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Corpus string `json:"corpus"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !corpusNamePattern.MatchString(req.Corpus) {
		jsonError(w, http.StatusBadRequest, "invalid corpus name")
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	corpusPath := filepath.Join(cfg.Corpus.Dir, req.Corpus)
	specs, err := corpus.Load(corpusPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "corpus: "+err.Error())
		return
	}
	if len(specs) == 0 {
		jsonError(w, http.StatusBadRequest, "corpus is empty")
		return
	}

	s.replayMu.Lock()
	if s.replayST.Running {
		s.replayMu.Unlock()
		jsonError(w, http.StatusConflict, "replay already running")
		return
	}
	s.replayST = replayStatus{Running: true, Corpus: req.Corpus, Total: len(specs)}
	s.replayMu.Unlock()

	go s.runReplay(cfg, specs, req.Corpus)
	writeJSON(w, map[string]any{"started": true, "total": len(specs)})
}

func (s *Server) runReplay(cfg *config.Config, specs []corpus.RequestSpec, corpusName string) {
	finish := func(reportPath, errMsg string) {
		s.replayMu.Lock()
		s.replayST.Running = false
		s.replayST.Current = ""
		if reportPath != "" {
			s.replayST.LastReport = filepath.Base(reportPath)
		}
		s.replayST.LastError = errMsg
		s.replayMu.Unlock()
	}

	runner := replay.NewRunner(cfg)
	runner.OnProgress = func(done, total int, name string) {
		s.replayMu.Lock()
		s.replayST.Done, s.replayST.Total, s.replayST.Current = done, total, name
		s.replayMu.Unlock()
	}
	if err := runner.Start(); err != nil {
		finish("", err.Error())
		return
	}
	defer runner.Stop()

	results := runner.Run(specs)
	path, err := report.WriteJSON(results, cfg.Report.Dir, corpusName, cfg.Old.BaseURL, cfg.New.BaseURL)
	if err != nil {
		finish("", err.Error())
		return
	}
	log.Printf("[replay] finished %s -> %s", corpusName, path)
	finish(path, "")
}

// -- helpers ------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
