// Package agent implements the collector-side link to the dashboard,
// Spring-Boot-Admin style: the collector registers itself OUTBOUND to the
// dashboard (works from private networks — no inbound holes needed), sends
// heartbeats with live status, and pushes the finished recording file.
// Without --dashboard the collector runs standalone (offline mode: move the
// file yourself or import it in the dashboard UI).
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Reporter struct {
	dashboard string
	token     string
	id        string
	client    *http.Client

	mu       sync.Mutex
	state    string // idle | capturing | done
	count    int
	lastErr  string
	stopped  chan struct{}
	stopOnce sync.Once
}

type registerReq struct {
	Name    string `json:"name"`
	Mode    string `json:"mode"`
	Version string `json:"version"`
	Out     string `json:"out"`
}

// Connect registers with the dashboard and starts the heartbeat loop.
// Returns nil (standalone mode) when dashboardURL is empty.
func Connect(dashboardURL, token, name, mode, out, version string) *Reporter {
	if dashboardURL == "" {
		return nil
	}
	r := &Reporter{
		dashboard: dashboardURL, token: token,
		client:  &http.Client{Timeout: 5 * time.Second},
		state:   "capturing",
		stopped: make(chan struct{}),
	}
	if name == "" {
		name, _ = os.Hostname()
	}
	body, _ := json.Marshal(registerReq{Name: name, Mode: mode, Version: version, Out: out})
	resp, err := r.post("/api/agents/register", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[agent] dashboard register failed (%v) — running standalone", err)
		return nil
	}
	defer resp.Body.Close()
	var reg struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil || reg.ID == "" {
		log.Printf("[agent] dashboard register failed (bad response) — running standalone")
		return nil
	}
	r.id = reg.ID
	log.Printf("[agent] registered with dashboard %s (id=%s)", dashboardURL, reg.ID)
	go r.heartbeatLoop()
	return r
}

func (r *Reporter) post(path, contentType string, body *bytes.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", r.dashboard+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if r.token != "" {
		req.Header.Set("X-MB-Token", r.token)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("dashboard returned %s", resp.Status)
	}
	return resp, nil
}

// Update refreshes the live counters shown on the dashboard.
func (r *Reporter) Update(count int, lastErr string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.count, r.lastErr = count, lastErr
	r.mu.Unlock()
}

func (r *Reporter) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	r.beat() // immediate first beat
	for {
		select {
		case <-r.stopped:
			return
		case <-ticker.C:
			r.beat()
		}
	}
}

func (r *Reporter) beat() {
	r.mu.Lock()
	payload, _ := json.Marshal(map[string]any{
		"id": r.id, "state": r.state, "count": r.count, "last_error": r.lastErr,
	})
	r.mu.Unlock()
	resp, err := r.post("/api/agents/heartbeat", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("[agent] heartbeat failed: %v", err)
		return
	}
	resp.Body.Close()
}

// Finish sends a final heartbeat and uploads the recording to the dashboard.
func (r *Reporter) Finish(recordingPath string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.state = "done"
	r.mu.Unlock()
	r.beat()
	r.stopOnce.Do(func() { close(r.stopped) })

	raw, err := os.ReadFile(recordingPath)
	if err != nil {
		log.Printf("[agent] upload skipped: %v", err)
		return
	}
	name := filepath.Base(recordingPath)
	resp, err := r.post("/api/agents/upload?id="+r.id+"&name="+name,
		"application/octet-stream", bytes.NewReader(raw))
	if err != nil {
		log.Printf("[agent] upload failed: %v — file remains at %s", err, recordingPath)
		return
	}
	resp.Body.Close()
	log.Printf("[agent] recording uploaded to dashboard: %s (%d bytes)", name, len(raw))
}
