package replay

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pajamasi726/mocking-box/internal/binlog"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/diff"
	"github.com/pajamasi726/mocking-box/internal/golden"
)

// Verifier replays a golden against the NEW stack only (Record & Verify mode):
// the golden's recorded responses/write-sets act as the old side.
type Verifier struct {
	cfg     *config.Config
	client  *http.Client
	capture *binlog.Capture

	OnProgress func(done, total int, name string)
}

func NewVerifier(cfg *config.Config) *Verifier {
	v := &Verifier{
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.HTTPTimeoutS * float64(time.Second))},
	}
	if cfg.New.MySQL != nil {
		v.capture = binlog.New("new", cfg.New.MySQL, 5502)
	}
	return v
}

func (v *Verifier) Start() error {
	if v.capture != nil {
		return v.capture.Start()
	}
	return nil
}

func (v *Verifier) Stop() {
	if v.capture != nil {
		v.capture.Stop()
	}
}

func (v *Verifier) Run(meta golden.Meta, entries []golden.Entry) []Result {
	sortRules := make([]diff.SortRule, len(v.cfg.Compare.SortArrays))
	for i, sr := range v.cfg.Compare.SortArrays {
		sortRules[i] = diff.SortRule{Path: sr.Path, By: sr.By}
	}
	// golden stores no response headers, so header comparison is skipped here
	opts := diff.Options{
		NoisePaths: v.cfg.Noise.ResponsePaths,
		SortArrays: sortRules,
	}

	results := make([]Result, 0, len(entries))
	for i, entry := range entries {
		log.Printf("(%d/%d) %s  %s %s", i+1, len(entries), entry.Name, entry.Method, entry.Path)
		if v.OnProgress != nil {
			v.OnProgress(i, len(entries), entry.Name)
		}
		results = append(results, v.verifyOne(entry, opts))
	}
	if v.OnProgress != nil {
		v.OnProgress(len(entries), len(entries), "")
	}
	return results
}

func (v *Verifier) verifyOne(entry golden.Entry, opts diff.Options) Result {
	started := time.Now()
	spec := entry.RequestSpec()
	result := Result{Name: entry.Name, Request: spec.Describe(), Verdict: diff.Error}
	defer func() { result.DurationMs = time.Since(started).Milliseconds() }()

	res, err := fireStack(v.client, v.cfg.New.BaseURL, v.capture, v.cfg.Attribution, spec)
	if err != nil {
		result.Error = fmt.Sprintf("new: %v", err)
		return result
	}

	// golden expected values play the "old" role in results/reports
	result.OldStatus, result.OldBody = entry.Expected.Status, entry.Expected.Body
	result.NewStatus, result.NewBody = res.status, res.body

	responseDiffs := diff.DiffResponses(
		entry.Expected.Status, entry.Expected.Body,
		res.status, res.body,
		nil, res.headers, opts,
	)

	newWS := orEmptyWS(diff.NormalizeWriteset(res.changes, v.cfg.Noise.Columns, v.cfg.Noise.TablesIgnore))
	result.NewWriteset = newWS

	var writesetDiffs []diff.Difference
	if entry.Expected.Writeset != nil {
		result.OldWriteset = entry.Expected.Writeset
		writesetDiffs = diff.DiffWritesets(entry.Expected.Writeset, newWS)
	} else {
		// capture couldn't attribute write-sets (concurrent traffic) — response-only verdict
		result.OldWriteset = []diff.WriteEntry{}
	}

	result.Differences = append(responseDiffs, writesetDiffs...)
	result.Verdict = diff.VerdictOf(responseDiffs, writesetDiffs)
	return result
}
