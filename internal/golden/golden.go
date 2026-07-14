// Package golden defines the portable golden artifact: a JSONL file holding
// captured requests plus their expected (recorded) responses and, when
// attributable, expected write-sets. A golden is everything Record & Verify
// needs to validate a new implementation without the old one running.
package golden

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/diff"
)

const Ext = ".golden.jsonl"

func IsGoldenFile(name string) bool { return strings.HasSuffix(name, Ext) }

type Meta struct {
	Type       string `json:"type"` // "meta"
	Version    int    `json:"version"`
	CreatedAt  string `json:"created_at"`
	Upstream   string `json:"upstream,omitempty"`
	Serialized bool   `json:"serialized"` // write-sets attributed per request?
	Source     string `json:"source,omitempty"`
}

type Expected struct {
	Status   int               `json:"status"`
	Body     string            `json:"body"`
	Writeset []diff.WriteEntry `json:"writeset"` // nil = not attributable at capture
}

type Entry struct {
	Type     string            `json:"type"` // "entry"
	Name     string            `json:"name"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     any               `json:"body,omitempty"`
	Expected Expected          `json:"expected"`
}

func (e Entry) RequestSpec() corpus.RequestSpec {
	return corpus.RequestSpec{
		Name: e.Name, Method: e.Method, Path: e.Path, Headers: e.Headers, Body: e.Body,
	}
}

// Writer appends golden lines to a file (meta first).
type Writer struct {
	f *os.File
}

func NewWriter(path string, meta Meta) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	w := &Writer{f: f}
	info, err := f.Stat()
	if err == nil && info.Size() == 0 {
		meta.Type = "meta"
		if meta.Version == 0 {
			meta.Version = 1
		}
		if meta.CreatedAt == "" {
			meta.CreatedAt = time.Now().Format(time.RFC3339)
		}
		if err := w.writeLine(meta); err != nil {
			f.Close()
			return nil, err
		}
	}
	return w, nil
}

func (w *Writer) Append(e Entry) error {
	e.Type = "entry"
	return w.writeLine(e)
}

func (w *Writer) writeLine(v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.f.Write(append(line, '\n'))
	return err
}

func (w *Writer) Close() error { return w.f.Close() }

// Read loads a golden file (meta + entries).
func Read(path string) (Meta, []Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, nil, err
	}
	defer f.Close()

	var meta Meta
	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &probe); err != nil {
			return meta, nil, fmt.Errorf("%s line %d: %w", path, lineNo, err)
		}
		switch probe.Type {
		case "meta":
			if err := json.Unmarshal([]byte(line), &meta); err != nil {
				return meta, nil, fmt.Errorf("%s line %d: %w", path, lineNo, err)
			}
		case "entry":
			var e Entry
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				return meta, nil, fmt.Errorf("%s line %d: %w", path, lineNo, err)
			}
			entries = append(entries, e)
		}
	}
	return meta, entries, scanner.Err()
}
