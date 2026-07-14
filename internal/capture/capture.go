// Package capture implements the recording reverse proxy: put it in front of
// a live server (usually the old stack), and every request that flows through
// is appended to a corpus JSONL file, ready for replay.
package capture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var skipHeaders = map[string]bool{
	"host": true, "content-length": true, "connection": true,
	"accept-encoding": true, "cookie": true,
}

// Recorder is a running capture session.
type Recorder struct {
	Listen     string
	Upstream   string
	CorpusPath string

	mu      sync.Mutex
	file    *os.File
	count   int
	started time.Time
	srv     *http.Server
	seq     int
	lastErr string
}

type Status struct {
	Running    bool   `json:"running"`
	Listen     string `json:"listen,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
	CorpusPath string `json:"corpus_path,omitempty"`
	Count      int    `json:"count"`
	StartedAt  string `json:"started_at,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

// Start launches a recording proxy: listen address -> upstream, appending
// captured requests to corpusPath (JSONL).
func Start(listen, upstream, corpusPath string) (*Recorder, error) {
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("invalid upstream URL %q", upstream)
	}
	file, err := os.OpenFile(corpusPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	r := &Recorder{Listen: listen, Upstream: upstream, CorpusPath: corpusPath, file: file, started: time.Now()}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		r.mu.Lock()
		r.lastErr = fmt.Sprintf("upstream %s: %v", upstream, err)
		r.mu.Unlock()
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "mocking-box capture: upstream unreachable: %s"}`, upstream)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		r.record(req, bodyBytes)
		proxy.ServeHTTP(w, req)
	})

	srv := &http.Server{Addr: listen, Handler: handler}
	r.srv = srv
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			r.mu.Lock()
			r.lastErr = err.Error()
			r.mu.Unlock()
		}
	}()
	// surface immediate bind failures (port already in use, …)
	select {
	case err := <-errCh:
		file.Close()
		return nil, err
	case <-time.After(150 * time.Millisecond):
	}
	log.Printf("[capture] recording %s -> %s into %s", listen, upstream, corpusPath)
	return r, nil
}

func (r *Recorder) record(req *http.Request, bodyBytes []byte) {
	headers := map[string]string{}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if skipHeaders[lk] {
			continue
		}
		headers[lk] = req.Header.Get(k)
	}

	var body any
	if len(bodyBytes) > 0 {
		var parsed any
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			body = parsed
		} else {
			body = string(bodyBytes)
		}
	}

	reqPath := req.URL.Path
	if req.URL.RawQuery != "" {
		reqPath += "?" + req.URL.RawQuery
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	entry := map[string]any{
		"name":   fmt.Sprintf("cap-%d-%s-%s", r.seq, req.Method, req.URL.Path),
		"method": req.Method,
		"path":   reqPath,
	}
	if len(headers) > 0 {
		entry["headers"] = headers
	}
	if body != nil {
		entry["body"] = body
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	if _, err := r.file.Write(append(line, '\n')); err != nil {
		r.lastErr = err.Error()
		return
	}
	r.count++
}

func (r *Recorder) Stop() {
	if r.srv != nil {
		r.srv.Close()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		r.file.Close()
		r.file = nil
	}
	log.Printf("[capture] stopped, %d request(s) recorded into %s", r.count, r.CorpusPath)
}

func (r *Recorder) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return Status{
		Running:    r.file != nil,
		Listen:     r.Listen,
		Upstream:   r.Upstream,
		CorpusPath: r.CorpusPath,
		Count:      r.count,
		StartedAt:  r.started.Format(time.RFC3339),
		LastError:  r.lastErr,
	}
}
