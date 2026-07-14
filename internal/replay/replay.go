// Package replay orchestrates sequential replay against both stacks.
package replay

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pajamasi726/mocking-box/internal/binlog"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/diff"
)

const maxBodyBytes = 1 << 20 // 1 MiB per response kept for the report

// Result is the outcome of replaying one request against both stacks.
type Result struct {
	Name        string            `json:"name"`
	Request     string            `json:"request"`
	Verdict     string            `json:"verdict"`
	Differences []diff.Difference `json:"differences"`
	OldStatus   int               `json:"old_status,omitempty"`
	NewStatus   int               `json:"new_status,omitempty"`
	OldBody     string            `json:"old_body,omitempty"`
	NewBody     string            `json:"new_body,omitempty"`
	OldWriteset []diff.WriteEntry `json:"old_writeset"`
	NewWriteset []diff.WriteEntry `json:"new_writeset"`
	Error       string            `json:"error,omitempty"`
	DurationMs  int64             `json:"duration_ms"`
}

type stackRuntime struct {
	cfg     config.Stack
	capture *binlog.Capture
}

type Runner struct {
	cfg    *config.Config
	client *http.Client
	old    *stackRuntime
	new    *stackRuntime

	// OnProgress, when set, is called before each request is replayed.
	OnProgress func(done, total int, name string)
}

func NewRunner(cfg *config.Config) *Runner {
	newRt := func(stack config.Stack, serverID uint32) *stackRuntime {
		rt := &stackRuntime{cfg: stack}
		if stack.MySQL != nil {
			rt.capture = binlog.New(stack.Name, stack.MySQL, serverID)
		}
		return rt
	}
	return &Runner{
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.HTTPTimeoutS * float64(time.Second))},
		old:    newRt(cfg.Old, 5501),
		new:    newRt(cfg.New, 5502),
	}
}

func (r *Runner) Start() error {
	for _, rt := range []*stackRuntime{r.old, r.new} {
		if rt.capture != nil {
			if err := rt.capture.Start(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) Stop() {
	for _, rt := range []*stackRuntime{r.old, r.new} {
		if rt.capture != nil {
			rt.capture.Stop()
		}
	}
}

func (r *Runner) Run(specs []corpus.RequestSpec) []Result {
	results := make([]Result, 0, len(specs))
	for i, spec := range specs {
		log.Printf("(%d/%d) %s  %s", i+1, len(specs), spec.Name, spec.Describe())
		if r.OnProgress != nil {
			r.OnProgress(i, len(specs), spec.Name)
		}
		results = append(results, r.runOne(spec))
	}
	if r.OnProgress != nil {
		r.OnProgress(len(specs), len(specs), "")
	}
	return results
}

type fired struct {
	status  int
	body    string
	headers map[string]string
	changes []binlog.RowChange
}

func (r *Runner) runOne(spec corpus.RequestSpec) Result {
	started := time.Now()
	result := Result{Name: spec.Name, Request: spec.Describe(), Verdict: diff.Error}
	defer func() { result.DurationMs = time.Since(started).Milliseconds() }()

	oldRes, err := r.fire(r.old, spec)
	if err != nil {
		result.Error = fmt.Sprintf("old: %v", err)
		return result
	}
	newRes, err := r.fire(r.new, spec)
	if err != nil {
		result.Error = fmt.Sprintf("new: %v", err)
		return result
	}

	result.OldStatus, result.NewStatus = oldRes.status, newRes.status
	result.OldBody, result.NewBody = oldRes.body, newRes.body

	sortRules := make([]diff.SortRule, len(r.cfg.Compare.SortArrays))
	for i, sr := range r.cfg.Compare.SortArrays {
		sortRules[i] = diff.SortRule{Path: sr.Path, By: sr.By}
	}
	responseDiffs := diff.DiffResponses(
		oldRes.status, oldRes.body,
		newRes.status, newRes.body,
		oldRes.headers, newRes.headers,
		diff.Options{
			NoisePaths:     r.cfg.Noise.ResponsePaths,
			CompareHeaders: r.cfg.CompareHeaders,
			SortArrays:     sortRules,
		},
	)

	result.OldWriteset = orEmptyWS(diff.NormalizeWriteset(oldRes.changes, r.cfg.Noise.Columns, r.cfg.Noise.TablesIgnore))
	result.NewWriteset = orEmptyWS(diff.NormalizeWriteset(newRes.changes, r.cfg.Noise.Columns, r.cfg.Noise.TablesIgnore))
	writesetDiffs := diff.DiffWritesets(result.OldWriteset, result.NewWriteset)

	result.Differences = append(responseDiffs, writesetDiffs...)
	result.Verdict = diff.VerdictOf(responseDiffs, writesetDiffs)
	return result
}

func (r *Runner) fire(rt *stackRuntime, spec corpus.RequestSpec) (*fired, error) {
	if rt.capture != nil {
		rt.capture.BeginWindow()
	}

	bodyBytes, isJSON, err := spec.BodyBytes()
	if err != nil {
		return nil, err
	}
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(spec.Method, rt.cfg.BaseURL+spec.Path, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if isJSON && req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	for k := range resp.Header {
		headers[toLower(k)] = resp.Header.Get(k)
	}

	out := &fired{status: resp.StatusCode, body: string(body), headers: headers}
	if rt.capture != nil {
		out.changes, err = rt.capture.TakeWindow(r.cfg.Attribution)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// orEmptyWS keeps empty write-sets as [] (not null) in report JSON.
func orEmptyWS(ws []diff.WriteEntry) []diff.WriteEntry {
	if ws == nil {
		return []diff.WriteEntry{}
	}
	return ws
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
