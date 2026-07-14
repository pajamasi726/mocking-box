// Package mcp is a Model Context Protocol server for mocking-box: it exposes
// the verification workflow as AI-callable tools over stdio JSON-RPC, so an
// assistant (Claude, Cursor, …) can drive setup and testing by conversation.
//
// It wraps a running dashboard's HTTP API. Safety: production databases are
// only ever read (test-connection, discover). Write operations (copy_db,
// verify) require an explicit confirm:true argument so an AI can't run them by
// accident — the human confirms in the conversation first.
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Server struct {
	dashboard string
	token     string
	client    *http.Client
	out       *bufio.Writer
}

func NewServer(dashboardURL, token string) *Server {
	return &Server{
		dashboard: dashboardURL, token: token,
		client: &http.Client{Timeout: 5 * time.Minute},
		out:    bufio.NewWriter(os.Stdout),
	}
}

// -- JSON-RPC framing (line-delimited) ---------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) reply(id json.RawMessage, result any, rpcErr *rpcError) {
	resp := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id)}
	if rpcErr != nil {
		resp["error"] = rpcErr
	} else {
		resp["result"] = result
	}
	line, _ := json.Marshal(resp)
	s.out.Write(line)
	s.out.WriteByte('\n')
	s.out.Flush()
}

// Run reads JSON-RPC requests from stdin and serves them until EOF.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		s.dispatch(req)
	}
	return scanner.Err()
}

func (s *Server) dispatch(req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.reply(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "mocking-box", "version": "0.5.0"},
		}, nil)
	case "notifications/initialized", "notifications/cancelled":
		// no response for notifications
	case "tools/list":
		s.reply(req.ID, map[string]any{"tools": toolSpecs}, nil)
	case "tools/call":
		s.callTool(req)
	case "ping":
		s.reply(req.ID, map[string]any{}, nil)
	default:
		if len(req.ID) > 0 {
			s.reply(req.ID, nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method})
		}
	}
}

// -- dashboard HTTP helpers ---------------------------------------------------

func (s *Server) get(path string) ([]byte, error) { return s.do("GET", path, nil) }
func (s *Server) postJSON(path string, body any) ([]byte, error) {
	raw, _ := json.Marshal(body)
	return s.do("POST", path, raw)
}
func (s *Server) putJSON(path string, body any) ([]byte, error) {
	raw, _ := json.Marshal(body)
	return s.do("PUT", path, raw)
}

func (s *Server) do(method, path string, body []byte) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, s.dashboard+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.token != "" {
		req.Header.Set("X-MB-Token", s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dashboard %s unreachable: %w", s.dashboard, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("dashboard returned %s: %s", resp.Status, string(data))
	}
	return data, nil
}
