package ui

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/golden"
	"github.com/pajamasi726/mocking-box/internal/pg"
)

// -- collector registry (Spring-Boot-Admin style: agents register inbound) ----

type agentRecord struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Mode          string           `json:"mode"`
	Version       string           `json:"version"`
	Out           string           `json:"out"`
	State         string           `json:"state"`
	Count         int              `json:"count"`
	Current       string           `json:"current,omitempty"`
	RemainSec     int              `json:"remain_sec"`
	Recordings    []map[string]any `json:"recordings,omitempty"`
	CaptureOK     bool             `json:"capture_ok"`
	CaptureDetail string           `json:"capture_detail"`
	Iface         string           `json:"iface"`
	LastError     string           `json:"last_error,omitempty"`
	Remote        string           `json:"remote"`
	FirstSeen     time.Time        `json:"first_seen"`
	LastSeen      time.Time        `json:"last_seen"`

	commands []map[string]any // queued commands handed back on next heartbeat
}

type agentRegistry struct {
	mu     sync.Mutex
	agents map[string]*agentRecord
	seq    int
}

func newAgentRegistry() *agentRegistry {
	return &agentRegistry{agents: map[string]*agentRecord{}}
}

func (s *Server) checkToken(r *http.Request) bool {
	return s.token == "" || r.Header.Get("X-MB-Token") == s.token
}

func (s *Server) agentRegister(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		jsonError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req struct {
		Name, Mode, Version, Out string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.registry.mu.Lock()
	s.registry.seq++
	id := fmt.Sprintf("agent-%d", s.registry.seq)
	s.registry.agents[id] = &agentRecord{
		ID: id, Name: req.Name, Mode: req.Mode, Version: req.Version, Out: req.Out,
		State: "capturing", Remote: r.RemoteAddr,
		FirstSeen: time.Now(), LastSeen: time.Now(),
	}
	s.registry.mu.Unlock()
	writeJSON(w, map[string]string{"id": id})
}

func (s *Server) agentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		jsonError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req struct {
		ID            string           `json:"id"`
		State         string           `json:"state"`
		Count         int              `json:"count"`
		Current       string           `json:"current"`
		RemainSec     int              `json:"remain_sec"`
		Recordings    []map[string]any `json:"recordings"`
		CaptureOK     bool             `json:"capture_ok"`
		CaptureDetail string           `json:"capture_detail"`
		Iface         string           `json:"iface"`
		LastError     string           `json:"last_error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	var commands []map[string]any
	s.registry.mu.Lock()
	if a, ok := s.registry.agents[req.ID]; ok {
		a.State, a.Count, a.LastError, a.LastSeen = req.State, req.Count, req.LastError, time.Now()
		a.Current, a.RemainSec, a.Recordings = req.Current, req.RemainSec, req.Recordings
		a.CaptureOK, a.CaptureDetail, a.Iface = req.CaptureOK, req.CaptureDetail, req.Iface
		commands = a.commands
		a.commands = nil // hand off queued commands exactly once
	}
	s.registry.mu.Unlock()
	if commands == nil {
		commands = []map[string]any{}
	}
	writeJSON(w, map[string]any{"ok": true, "commands": commands})
}

func (s *Server) agentList(w http.ResponseWriter, r *http.Request) {
	s.registry.mu.Lock()
	out := make([]map[string]any, 0, len(s.registry.agents))
	for _, a := range s.registry.agents {
		fresh := time.Since(a.LastSeen)
		health := "green"
		switch {
		case fresh > 2*time.Minute:
			health = "red"
		case fresh > 30*time.Second:
			health = "yellow"
		}
		out = append(out, map[string]any{
			"id": a.ID, "name": a.Name, "mode": a.Mode, "state": a.State,
			"count": a.Count, "last_error": a.LastError, "out": a.Out,
			"current": a.Current, "remain_sec": a.RemainSec, "recordings": a.Recordings,
			"capture_ok": a.CaptureOK, "capture_detail": a.CaptureDetail, "iface": a.Iface,
			"last_seen_s": int(fresh.Seconds()), "health": health,
		})
	}
	s.registry.mu.Unlock()
	writeJSON(w, out)
}

// agentCommand queues a control command (start/stop/upload) for an agent; the
// agent picks it up on its next heartbeat.
func (s *Server) agentCommand(w http.ResponseWriter, r *http.Request) {
	var cmd map[string]any
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, _ := cmd["id"].(string)
	s.registry.mu.Lock()
	a, ok := s.registry.agents[id]
	if ok {
		delete(cmd, "id")
		a.commands = append(a.commands, cmd)
	}
	s.registry.mu.Unlock()
	if !ok {
		jsonError(w, http.StatusNotFound, "unknown agent")
		return
	}
	writeJSON(w, map[string]bool{"queued": true})
}

// agentUpload receives a finished recording pushed by a collector.
func (s *Server) agentUpload(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		jsonError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	s.saveRecording(w, r, r.URL.Query().Get("name"))
}

// corpusUpload is the offline-mode import from the dashboard UI.
func (s *Server) corpusUpload(w http.ResponseWriter, r *http.Request) {
	s.saveRecording(w, r, r.URL.Query().Get("name"))
}

func (s *Server) saveRecording(w http.ResponseWriter, r *http.Request, name string) {
	name = filepath.Base(strings.TrimSpace(name))
	okExt := strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".har")
	if !corpusNamePattern.MatchString(name) || !okExt {
		jsonError(w, http.StatusBadRequest, "invalid recording name (.jsonl/.golden.jsonl/.har)")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512<<20))
	if err != nil || len(raw) == 0 {
		jsonError(w, http.StatusBadRequest, "empty body")
		return
	}
	dir := s.corpusDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"saved": name, "bytes": len(raw), "golden": golden.IsGoldenFile(name)})
}

// -- connection health (green/red with a reason) ------------------------------

type healthItem struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	out := map[string]healthItem{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	add := func(key string, fn func() healthItem) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item := fn()
			mu.Lock()
			out[key] = item
			mu.Unlock()
		}()
	}

	if cfg.Old.BaseURL != "" { // old = 선택 (라이브 비교/기준선용)
		add("old_api", func() healthItem { return checkHTTP(cfg.Old.BaseURL) })
	}
	add("new_api", func() healthItem { return checkHTTP(cfg.New.BaseURL) })
	for _, d := range cfg.New.Datastores {
		ds := d
		add("db:"+ds.Name, func() healthItem { return checkDatastore(ds) })
	}
	wg.Wait()
	writeJSON(w, out)
}

func checkHTTP(baseURL string) healthItem {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		return healthItem{false, err.Error()}
	}
	resp.Body.Close()
	return healthItem{true, fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func checkDatastore(d *config.Datastore) healthItem {
	if d.Type == "postgres" {
		ok, detail := pg.Health(d)
		return healthItem{ok, detail}
	}
	return checkMySQL(d.MySQL())
}

func checkMySQL(m *config.MySQL) healthItem {
	db, err := sql.Open("mysql", m.DSN()+"?timeout=3s")
	if err != nil {
		return healthItem{false, err.Error()}
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Second)
	if err := db.Ping(); err != nil {
		return healthItem{false, "connect: " + err.Error()}
	}
	var format string
	if err := db.QueryRow("SELECT @@binlog_format").Scan(&format); err != nil {
		return healthItem{false, "binlog_format 조회 실패: " + err.Error()}
	}
	if format != "ROW" {
		return healthItem{false, "binlog_format=" + format + " (ROW 필요 — write-set 캡처 불가)"}
	}
	return healthItem{true, "connected · binlog_format=ROW"}
}
