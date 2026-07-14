// Package corpus loads request corpora: JSONL (native) and HAR captures.
package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RequestSpec is one replayable HTTP request.
type RequestSpec struct {
	Name    string            `json:"name"`
	Ts      string            `json:"ts,omitempty"` // request arrival time for load-time ordering
	Method  string            `json:"method"`
	Path    string            `json:"path"` // path (+ query), joined onto each stack's base_url
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"` // map/[]any -> JSON, string -> raw, nil -> none
}

func (r RequestSpec) Describe() string { return r.Method + " " + r.Path }

// BodyBytes renders the body and reports whether it is JSON.
func (r RequestSpec) BodyBytes() (data []byte, isJSON bool, err error) {
	switch b := r.Body.(type) {
	case nil:
		return nil, false, nil
	case string:
		return []byte(b), false, nil
	default:
		data, err = json.Marshal(b)
		return data, true, err
	}
}

func Load(path string) ([]RequestSpec, error) {
	var specs []RequestSpec
	var err error
	if strings.EqualFold(filepath.Ext(path), ".har") {
		specs, err = loadHAR(path)
	} else {
		specs, err = loadJSONL(path)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Ts < specs[j].Ts })
	return specs, nil
}

func loadJSONL(path string) ([]RequestSpec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var specs []RequestSpec
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var spec RequestSpec
		if err := json.Unmarshal([]byte(line), &spec); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, lineNo, err)
		}
		if spec.Name == "" {
			spec.Name = fmt.Sprintf("req-%d", lineNo)
		}
		spec.Method = strings.ToUpper(spec.Method)
		lower := map[string]string{}
		for k, v := range spec.Headers {
			lower[strings.ToLower(k)] = v
		}
		spec.Headers = lower
		specs = append(specs, spec)
	}
	return specs, scanner.Err()
}

var harSkipHeaders = map[string]bool{
	"host": true, "content-length": true, "cookie": true,
	"connection": true, "accept-encoding": true,
}

func loadHAR(path string) ([]RequestSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var har struct {
		Log struct {
			Entries []struct {
				Request struct {
					Method  string `json:"method"`
					URL     string `json:"url"`
					Headers []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"headers"`
					PostData *struct {
						MimeType string `json:"mimeType"`
						Text     string `json:"text"`
					} `json:"postData"`
				} `json:"request"`
			} `json:"entries"`
		} `json:"log"`
	}
	if err := json.Unmarshal(raw, &har); err != nil {
		return nil, fmt.Errorf("parse HAR %s: %w", path, err)
	}

	var specs []RequestSpec
	for i, entry := range har.Log.Entries {
		req := entry.Request
		u, err := url.Parse(req.URL)
		if err != nil {
			continue
		}
		reqPath := u.Path
		if u.RawQuery != "" {
			reqPath += "?" + u.RawQuery
		}
		headers := map[string]string{}
		for _, h := range req.Headers {
			name := strings.ToLower(h.Name)
			if harSkipHeaders[name] || strings.HasPrefix(name, ":") {
				continue
			}
			headers[name] = h.Value
		}
		var body any
		if req.PostData != nil && req.PostData.Text != "" {
			if strings.Contains(req.PostData.MimeType, "json") {
				var parsed any
				if json.Unmarshal([]byte(req.PostData.Text), &parsed) == nil {
					body = parsed
				} else {
					body = req.PostData.Text
				}
			} else {
				body = req.PostData.Text
			}
		}
		specs = append(specs, RequestSpec{
			Name:    fmt.Sprintf("har-%d-%s-%s", i+1, req.Method, u.Path),
			Method:  strings.ToUpper(req.Method),
			Path:    reqPath,
			Headers: headers,
			Body:    body,
		})
	}
	return specs, nil
}
