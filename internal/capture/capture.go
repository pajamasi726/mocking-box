// Package capture implements the recording reverse proxy: put it in front of
// a live server (usually the old stack), and every request that flows through
// is appended to a corpus, ready for replay.
//
// Two recording modes:
//   - requests-only: corpus JSONL (replay needs both stacks — live mode)
//   - golden: request + observed response (+ write-set when serialized) —
//     everything Record & Verify needs to test the new stack alone.
package capture

import (
	"bytes"
	"context"
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

	"github.com/pajamasi726/mocking-box/internal/binlog"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/diff"
	"github.com/pajamasi726/mocking-box/internal/golden"
)

const maxBodyBytes = 1 << 20

var skipHeaders = map[string]bool{
	"host": true, "content-length": true, "connection": true,
	"accept-encoding": true, "cookie": true,
}

// Options selects the recording mode.
type Options struct {
	Golden bool // record responses (golden artifact) instead of requests-only
	// Serialize handles one request at a time so binlog write-sets can be
	// attributed per request. Implied when Source is set.
	Serialize    bool
	Source       *config.MySQL // upstream's DB for expected write-sets (golden mode)
	Attribution  config.Attribution
	NoiseColumns []string
	TablesIgnore []string
}

type Status struct {
	Running    bool   `json:"running"`
	Listen     string `json:"listen,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
	CorpusPath string `json:"corpus_path,omitempty"`
	Golden     bool   `json:"golden"`
	Count      int    `json:"count"`
	StartedAt  string `json:"started_at,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type exchangeKey struct{}

type exchange struct {
	status int
	body   string
}

// Recorder is a running capture session.
type Recorder struct {
	Listen     string
	Upstream   string
	CorpusPath string
	opts       Options

	mu      sync.Mutex
	seq     int
	count   int
	started time.Time
	lastErr string

	serMu     sync.Mutex // serializes proxied requests in golden+source mode
	srv       *http.Server
	rawFile   *os.File       // requests-only mode
	goldenW   *golden.Writer // golden mode
	dbCapture *binlog.Capture
}

// Start launches a recording proxy on listen -> upstream.
func Start(listen, upstream, corpusPath string, opts Options) (*Recorder, error) {
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("invalid upstream URL %q", upstream)
	}
	if opts.Source != nil {
		opts.Serialize = true
	}

	r := &Recorder{
		Listen: listen, Upstream: upstream, CorpusPath: corpusPath,
		opts: opts, started: time.Now(),
	}

	if opts.Golden {
		w, err := golden.NewWriter(corpusPath, golden.Meta{
			Upstream:   upstream,
			Serialized: opts.Serialize && opts.Source != nil,
		})
		if err != nil {
			return nil, err
		}
		r.goldenW = w
	} else {
		f, err := os.OpenFile(corpusPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		r.rawFile = f
	}

	if opts.Source != nil {
		r.dbCapture = binlog.New("capture", opts.Source, 5599)
		if err := r.dbCapture.Start(); err != nil {
			r.closeOutputs()
			return nil, fmt.Errorf("write-set source: %w", err)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		r.setErr(fmt.Sprintf("upstream %s: %v", upstream, err))
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "mocking-box capture: upstream unreachable: %s"}`, upstream)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		ex, _ := resp.Request.Context().Value(exchangeKey{}).(*exchange)
		if ex == nil {
			return nil
		}
		buf, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		if err != nil {
			return err
		}
		rest := resp.Body
		resp.Body = readCloser{io.MultiReader(bytes.NewReader(buf), rest), rest}
		ex.status = resp.StatusCode
		ex.body = string(buf)
		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		if !r.opts.Golden {
			r.recordRequest(req, bodyBytes)
			proxy.ServeHTTP(w, req)
			return
		}

		if r.opts.Serialize {
			r.serMu.Lock()
			defer r.serMu.Unlock()
		}
		if r.dbCapture != nil {
			r.dbCapture.BeginWindow()
		}
		ex := &exchange{}
		proxy.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), exchangeKey{}, ex)))

		var ws []diff.WriteEntry
		if r.dbCapture != nil {
			changes, err := r.dbCapture.TakeWindow(r.opts.Attribution)
			if err != nil {
				r.setErr(err.Error())
			} else {
				ws = diff.NormalizeWriteset(changes, r.opts.NoiseColumns, r.opts.TablesIgnore)
				if ws == nil {
					ws = []diff.WriteEntry{}
				}
			}
		}
		r.recordGolden(req, bodyBytes, ex, ws)
	})

	srv := &http.Server{Addr: listen, Handler: handler}
	r.srv = srv
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			r.setErr(err.Error())
		}
	}()
	select {
	case err := <-errCh:
		r.closeOutputs()
		if r.dbCapture != nil {
			r.dbCapture.Stop()
		}
		return nil, err
	case <-time.After(150 * time.Millisecond):
	}
	mode := "requests"
	if opts.Golden {
		mode = "golden"
	}
	log.Printf("[capture] recording (%s) %s -> %s into %s", mode, listen, upstream, corpusPath)
	return r, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

func requestParts(req *http.Request, bodyBytes []byte) (path string, headers map[string]string, body any) {
	headers = map[string]string{}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if skipHeaders[lk] {
			continue
		}
		headers[lk] = req.Header.Get(k)
	}
	if len(bodyBytes) > 0 {
		var parsed any
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			body = parsed
		} else {
			body = string(bodyBytes)
		}
	}
	path = req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}
	return path, headers, body
}

func (r *Recorder) nextName(req *http.Request) string {
	r.seq++
	return fmt.Sprintf("cap-%d-%s-%s", r.seq, req.Method, req.URL.Path)
}

func (r *Recorder) recordRequest(req *http.Request, bodyBytes []byte) {
	path, headers, body := requestParts(req, bodyBytes)
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := map[string]any{
		"name": r.nextName(req), "method": req.Method, "path": path,
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
	if _, err := r.rawFile.Write(append(line, '\n')); err != nil {
		r.lastErr = err.Error()
		return
	}
	r.count++
}

func (r *Recorder) recordGolden(req *http.Request, bodyBytes []byte, ex *exchange, ws []diff.WriteEntry) {
	path, headers, body := requestParts(req, bodyBytes)
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.goldenW.Append(golden.Entry{
		Name: r.nextName(req), Method: req.Method, Path: path,
		Headers: headers, Body: body,
		Expected: golden.Expected{Status: ex.status, Body: ex.body, Writeset: ws},
	})
	if err != nil {
		r.lastErr = err.Error()
		return
	}
	r.count++
}

func (r *Recorder) setErr(msg string) {
	r.mu.Lock()
	r.lastErr = msg
	r.mu.Unlock()
}

func (r *Recorder) closeOutputs() {
	if r.rawFile != nil {
		r.rawFile.Close()
		r.rawFile = nil
	}
	if r.goldenW != nil {
		r.goldenW.Close()
		r.goldenW = nil
	}
}

func (r *Recorder) Stop() {
	if r.srv != nil {
		r.srv.Close()
	}
	if r.dbCapture != nil {
		r.dbCapture.Stop()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeOutputs()
	log.Printf("[capture] stopped, %d request(s) recorded into %s", r.count, r.CorpusPath)
}

func (r *Recorder) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return Status{
		Running:    r.rawFile != nil || r.goldenW != nil,
		Listen:     r.Listen,
		Upstream:   r.Upstream,
		CorpusPath: r.CorpusPath,
		Golden:     r.opts.Golden,
		Count:      r.count,
		StartedAt:  r.started.Format(time.RFC3339),
		LastError:  r.lastErr,
	}
}
