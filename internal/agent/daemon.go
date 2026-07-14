package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/golden"
	"github.com/pajamasi726/mocking-box/internal/sniff"
)

// Command is an instruction the dashboard hands back on a heartbeat.
type Command struct {
	Action      string  `json:"action"` // "start" | "stop" | "upload"
	Mode        string  `json:"mode"`   // sniff | mirror | proxy
	Iface       string  `json:"iface"`
	Port        int     `json:"port"`
	Listen      string  `json:"listen"`
	Upstream    string  `json:"upstream"`
	Golden      bool    `json:"golden"`
	Sample      float64 `json:"sample"`
	Name        string  `json:"name"`         // recording file name
	DurationMin int     `json:"duration_min"` // auto-stop after N minutes (0 = manual only)
}

// Daemon is a long-running collector controlled by the dashboard: it registers,
// heartbeats, and executes start/stop/upload commands. Recordings are written
// to a local directory (streaming) and uploaded on demand.
type Daemon struct {
	r         *Reporter
	dir       string
	cfg       *config.Config // reserved for future proxy-golden write-set source
	ifaceHint string

	mu       sync.Mutex
	capName  string
	output   *sniff.Output
	stopCap  func()
	running  bool
	stopAt   time.Time // scheduled auto-stop (zero = none)
	autoStop *time.Timer
	lastErr  string

	captureOK     bool
	captureDetail string
	iface         string
}

// hardCapMinutes bounds any capture even when none is requested, so a forgotten
// or disconnected agent never records forever (disk safety).
const hardCapMinutes = 24 * 60

// RunDaemon connects to the dashboard and serves commands until ctx is done.
func RunDaemon(ctx context.Context, dashboardURL, token, name, dir string, cfg *config.Config, defaultIface string) error {
	if dashboardURL == "" {
		return fmt.Errorf("--dashboard is required for agent mode")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	r := &Reporter{
		dashboard: dashboardURL, token: token,
		client: &http.Client{Timeout: 30 * time.Second}, state: "idle",
	}
	if name == "" {
		name, _ = os.Hostname()
	}
	body, _ := json.Marshal(registerReq{Name: name, Mode: "agent", Version: "daemon", Out: dir})
	resp, err := r.post("/api/agents/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	var reg struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&reg)
	resp.Body.Close()
	r.id = reg.ID
	log.Printf("[agent] daemon registered (id=%s) dir=%s", reg.ID, dir)

	d := &Daemon{r: r, dir: dir, cfg: cfg, ifaceHint: defaultIface}
	// preflight: can this host capture packets? shown on the dashboard up-front,
	// so you know before starting whether CAP_NET_RAW is missing.
	d.captureOK, d.captureDetail = sniff.CanCapture(defaultIface)
	log.Printf("[agent] packet capture: ok=%v (%s)", d.captureOK, d.captureDetail)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			d.stopCapture()
			return nil
		case <-ticker.C:
			d.beat()
		}
	}
}

func (d *Daemon) beat() {
	d.mu.Lock()
	state, capName := "idle", d.capName
	if d.running {
		state = "capturing"
	}
	count := 0
	if d.output != nil {
		count, _ = d.output.Stats()
	}
	remainSec := 0
	if d.running && !d.stopAt.IsZero() {
		remainSec = int(time.Until(d.stopAt).Seconds())
	}
	lastErr := d.lastErr
	d.mu.Unlock()

	payload, _ := json.Marshal(map[string]any{
		"id": d.r.id, "state": state, "count": count, "last_error": lastErr,
		"current": capName, "remain_sec": remainSec, "recordings": d.listRecordings(),
		"capture_ok": d.captureOK, "capture_detail": d.captureDetail, "iface": d.ifaceHint,
	})
	resp, err := d.r.post("/api/agents/heartbeat", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("[agent] heartbeat: %v", err)
		return
	}
	var reply struct {
		Commands []Command `json:"commands"`
	}
	json.NewDecoder(resp.Body).Decode(&reply)
	resp.Body.Close()
	for _, cmd := range reply.Commands {
		d.exec(cmd)
	}
}

func (d *Daemon) listRecordings() []map[string]any {
	entries, _ := os.ReadDir(d.dir)
	out := []map[string]any{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".jsonl") && !strings.HasSuffix(n, ".har") {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		out = append(out, map[string]any{"name": n, "size": size, "golden": golden.IsGoldenFile(n)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
	return out
}

func (d *Daemon) exec(cmd Command) {
	switch cmd.Action {
	case "start":
		d.startCapture(cmd)
	case "stop":
		d.stopCapture()
	case "upload":
		d.upload(cmd.Name)
	}
}

func (d *Daemon) startCapture(cmd Command) {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	name := filepath.Base(cmd.Name)
	if name == "" || name == "." {
		name = "capture-" + time.Now().Format("20060102-150405") + ".golden.jsonl"
	}
	path := filepath.Join(d.dir, name)
	output, err := sniff.NewOutput(path, cmd.Mode+"://"+d.ifaceHint)
	if err != nil {
		d.mu.Unlock()
		log.Printf("[agent] start: %v", err)
		return
	}
	output.SampleRate, output.SampleAllWrites = cmd.Sample, true
	ctx, cancel := context.WithCancel(context.Background())

	mode := cmd.Mode
	iface := cmd.Iface
	if iface == "" {
		iface = d.ifaceHint
	}
	go func() {
		var runErr error
		switch mode {
		case "mirror":
			listen := cmd.Listen
			if listen == "" {
				listen = ":4789"
			}
			pipeline := sniff.NewPipeline(cmd.Port, output.Sink)
			runErr = sniff.RunMirror(listen, cmd.Port, pipeline, ctx.Done())
		default: // sniff
			pipeline := sniff.NewPipeline(cmd.Port, output.Sink)
			runErr = sniff.RunLive(iface, cmd.Port, pipeline, ctx.Done())
		}
		output.Close()
		if runErr != nil {
			// surface start failures (e.g. missing CAP_NET_RAW) to the dashboard
			log.Printf("[agent] capture ended: %v", runErr)
			d.mu.Lock()
			d.lastErr = runErr.Error()
			d.running = false
			d.stopAt = time.Time{}
			if d.autoStop != nil {
				d.autoStop.Stop()
				d.autoStop = nil
			}
			d.mu.Unlock()
		}
	}()

	d.capName = name
	d.output = output
	d.stopCap = cancel
	d.running = true
	d.lastErr = ""

	// auto-stop safety: honor the requested duration, else the hard cap
	mins := cmd.DurationMin
	if mins <= 0 || mins > hardCapMinutes {
		mins = hardCapMinutes
	}
	d.stopAt = time.Now().Add(time.Duration(mins) * time.Minute)
	d.autoStop = time.AfterFunc(time.Duration(mins)*time.Minute, func() {
		log.Printf("[agent] auto-stop after %d min: %s", mins, name)
		d.stopCapture()
	})
	d.mu.Unlock()
	log.Printf("[agent] capture started: %s (%s port %d, auto-stop in %dmin) -> %s", name, mode, cmd.Port, mins, path)
}

func (d *Daemon) stopCapture() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		return
	}
	if d.autoStop != nil {
		d.autoStop.Stop()
		d.autoStop = nil
	}
	if d.stopCap != nil {
		d.stopCap()
	}
	d.running = false
	d.stopAt = time.Time{}
	log.Printf("[agent] capture stopped: %s", d.capName)
}

func (d *Daemon) upload(name string) {
	name = filepath.Base(name)
	raw, err := os.ReadFile(filepath.Join(d.dir, name))
	if err != nil {
		log.Printf("[agent] upload %s: %v", name, err)
		return
	}
	resp, err := d.r.post("/api/agents/upload?id="+d.r.id+"&name="+name,
		"application/octet-stream", bytes.NewReader(raw))
	if err != nil {
		log.Printf("[agent] upload %s: %v", name, err)
		return
	}
	resp.Body.Close()
	log.Printf("[agent] uploaded %s (%d bytes)", name, len(raw))
}
