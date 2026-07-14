package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// toolSpecs is the tool catalog advertised to the AI client.
var toolSpecs = []map[string]any{
	{
		"name":        "get_config",
		"description": "Read the current mocking-box config (target servers, databases, noise/compare rules). Call this first to see what's set.",
		"inputSchema": obj(nil, nil),
	},
	{
		"name":        "test_connection",
		"description": "Check a database connection READ-ONLY without saving it (validates host/port/user/password). Safe on production — only connects and checks binlog_format/wal_level. Use before configuring a prod DB.",
		"inputSchema": obj(map[string]any{
			"type":     str("mysql | postgres"),
			"host":     str("DB host"),
			"port":     num("DB port (3306 mysql / 5432 postgres)"),
			"user":     str("DB user"),
			"password": str("DB password"),
			"database": str("database name (postgres only)"),
		}, []string{"type", "host", "user"}),
	},
	{
		"name":        "set_config",
		"description": "Fill in the verification config. Provide the new (under-test) and optionally old (reference/copy-source) stacks with their datastores. This is the 'fill in the connection info' step — an AI can populate it from what the user describes. Does not touch any DB.",
		"inputSchema": obj(map[string]any{
			"config": objRaw("Full config object (same shape as get_config returns): {new:{base_url,datastores:[...]}, old:{...}, noise:{...}, compare:{...}}"),
		}, []string{"config"}),
	},
	{
		"name":        "health",
		"description": "Test all configured connections (APIs + databases) and report green/red with the reason. Read-only.",
		"inputSchema": obj(nil, nil),
	},
	{
		"name":        "discover_db",
		"description": "List a database's schemas and tables with row counts and sizes (READ-ONLY, safe on prod). Use this to see what exists before copying — e.g. to spot huge tables to exclude.",
		"inputSchema": obj(map[string]any{
			"type": str("mysql | postgres"), "host": str("DB host"), "port": num("DB port"),
			"user": str("DB user"), "password": str("DB password"), "database": str("database (postgres)"),
		}, []string{"type", "host", "user"}),
	},
	{
		"name":        "copy_db",
		"description": "WRITE OPERATION. Copy schemas+data from the reference (old) DB into the verification (new) DB, per the saved config. Reads prod (SELECT only) and writes the isolated copy. Requires confirm:true — the human must approve first because it moves data.",
		"inputSchema": obj(map[string]any{
			"confirm": boolean("Must be true to run. Copies data into the verification DB."),
		}, []string{"confirm"}),
	},
	{
		"name":        "list_recordings",
		"description": "List captured traffic recordings (corpora) available to verify against.",
		"inputSchema": obj(nil, nil),
	},
	{
		"name":        "verify",
		"description": "WRITE OPERATION on the verification DB. Replay a recording against the new stack and diff responses + DB write-sets. Requires confirm:true. Returns when finished with a summary; use get_results for detail.",
		"inputSchema": obj(map[string]any{
			"corpus":  str("recording file name (from list_recordings)"),
			"confirm": boolean("Must be true to run."),
		}, []string{"corpus", "confirm"}),
	},
	{
		"name":        "list_runs",
		"description": "List past verification runs (most recent first) with their pass/diff summary.",
		"inputSchema": obj(nil, nil),
	},
	{
		"name":        "get_results",
		"description": "Get per-request results of a verification run (differences, verdicts). Omit 'file' for the most recent run.",
		"inputSchema": obj(map[string]any{
			"file":       str("run report file (from list_runs); omit for latest"),
			"only_diffs": boolean("if true, return only non-MATCH results"),
		}, nil),
	},
}

func (s *Server) callTool(req rpcRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	json.Unmarshal(req.Params, &params)
	args := map[string]any{}
	json.Unmarshal(params.Arguments, &args)

	text, err := s.runTool(params.Name, args, params.Arguments)
	if err != nil {
		s.reply(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
			"isError": true,
		}, nil)
		return
	}
	s.reply(req.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}, nil)
}

func (s *Server) runTool(name string, args map[string]any, raw json.RawMessage) (string, error) {
	switch name {
	case "get_config":
		return s.pretty(s.get("/api/config"))
	case "test_connection":
		return s.pretty(s.postJSON("/api/test-connection", args))
	case "discover_db":
		return s.pretty(s.postJSON("/api/seed/discover", args))
	case "set_config":
		cfg, ok := args["config"]
		if !ok {
			return "", fmt.Errorf("config argument required")
		}
		return s.pretty(s.putJSON("/api/config", cfg))
	case "health":
		return s.pretty(s.get("/api/health"))
	case "list_recordings":
		return s.pretty(s.get("/api/corpora"))
	case "list_runs":
		return s.pretty(s.get("/api/runs"))
	case "copy_db":
		if args["confirm"] != true {
			return "", fmt.Errorf("copy_db needs confirm:true — this copies data into the verification DB. Confirm with the user first.")
		}
		if _, err := s.postJSON("/api/seed/start", map[string]any{}); err != nil {
			return "", err
		}
		return s.pollSeed()
	case "verify":
		if args["confirm"] != true {
			return "", fmt.Errorf("verify needs confirm:true")
		}
		corpus, _ := args["corpus"].(string)
		if corpus == "" {
			return "", fmt.Errorf("corpus required (see list_recordings)")
		}
		if _, err := s.postJSON("/api/replay/start", map[string]any{"corpus": corpus}); err != nil {
			return "", err
		}
		return s.pollReplay()
	case "get_results":
		file, _ := args["file"].(string)
		if file == "" {
			runs, err := s.get("/api/runs")
			if err != nil {
				return "", err
			}
			var list []map[string]any
			json.Unmarshal(runs, &list)
			if len(list) == 0 {
				return "no runs yet", nil
			}
			file, _ = list[0]["file"].(string)
		}
		data, err := s.get("/api/run?file=" + url.QueryEscape(file))
		if err != nil {
			return "", err
		}
		return s.summarizeRun(data, args["only_diffs"] == true)
	}
	return "", fmt.Errorf("unknown tool %q", name)
}

// -- polling long ops ---------------------------------------------------------

func (s *Server) pollSeed() (string, error) {
	for i := 0; i < 600; i++ {
		data, err := s.get("/api/seed/status")
		if err != nil {
			return "", err
		}
		var st struct {
			Running bool
			Done    bool
			Error   string
			Tables  int
			Rows    int64
		}
		json.Unmarshal(data, &st)
		if !st.Running {
			if st.Error != "" {
				return "", errors.New(st.Error)
			}
			return fmt.Sprintf("copy complete: %d tables, %d rows into the verification DB", st.Tables, st.Rows), nil
		}
		time.Sleep(2 * time.Second)
	}
	return "copy still running (timed out waiting)", nil
}

func (s *Server) pollReplay() (string, error) {
	for i := 0; i < 900; i++ {
		data, err := s.get("/api/replay/status")
		if err != nil {
			return "", err
		}
		var st struct {
			Running    bool
			LastReport string `json:"last_report"`
			LastError  string `json:"last_error"`
			Done       int
			Total      int
		}
		json.Unmarshal(data, &st)
		if !st.Running {
			if st.LastError != "" {
				return "", errors.New(st.LastError)
			}
			if st.LastReport != "" {
				rep, _ := s.get("/api/run?file=" + url.QueryEscape(st.LastReport))
				return s.summarizeRun(rep, true)
			}
			return "verification finished", nil
		}
		time.Sleep(1 * time.Second)
	}
	return "verification still running (timed out)", nil
}

func (s *Server) summarizeRun(data []byte, onlyDiffs bool) (string, error) {
	var run struct {
		Corpus  string         `json:"corpus"`
		Summary map[string]int `json:"summary"`
		Results []struct {
			Name        string `json:"name"`
			Request     string `json:"request"`
			Verdict     string `json:"verdict"`
			Differences []struct {
				Kind string `json:"kind"`
				Path string `json:"path"`
				Old  any    `json:"old"`
				New  any    `json:"new"`
			} `json:"differences"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &run); err != nil {
		return string(data), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "corpus: %s\nsummary: %v\n", run.Corpus, run.Summary)
	for _, r := range run.Results {
		if onlyDiffs && r.Verdict == "MATCH" {
			continue
		}
		fmt.Fprintf(&b, "- [%s] %s %s", r.Verdict, r.Name, r.Request)
		if len(r.Differences) > 0 {
			d := r.Differences[0]
			fmt.Fprintf(&b, "  | %s: %v → %v", d.Path, d.Old, d.New)
			if len(r.Differences) > 1 {
				fmt.Fprintf(&b, " (+%d more)", len(r.Differences)-1)
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (s *Server) pretty(data []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}
	var v any
	if json.Unmarshal(data, &v) != nil {
		return string(data), nil
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	return string(out), nil
}

// -- schema helpers -----------------------------------------------------------

func obj(props map[string]any, required []string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	m := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}
func objRaw(desc string) map[string]any { return map[string]any{"type": "object", "description": desc} }
func str(desc string) map[string]any    { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any    { return map[string]any{"type": "integer", "description": desc} }
func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
