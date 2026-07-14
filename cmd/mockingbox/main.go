// mockingbox: universal differential-testing box.
//
//	mockingbox run -c config.yaml --corpus corpus.jsonl
//	mockingbox ui  --report-dir ./report [--addr :8642]
package main

import (
	"flag"
	"fmt"
	"log"
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pajamasi726/mocking-box/internal/agent"
	"github.com/pajamasi726/mocking-box/internal/capture"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/diff"
	"github.com/pajamasi726/mocking-box/internal/golden"
	"github.com/pajamasi726/mocking-box/internal/mcp"
	"github.com/pajamasi726/mocking-box/internal/replay"
	"github.com/pajamasi726/mocking-box/internal/report"
	"github.com/pajamasi726/mocking-box/internal/seed"
	"github.com/pajamasi726/mocking-box/internal/sniff"
	"github.com/pajamasi726/mocking-box/internal/ui"
	"github.com/pajamasi726/mocking-box/internal/wscapture"
)

func main() {
	log.SetFlags(log.Ltime)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	// 분석기 (dashboard/verifier)
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "verify":
		os.Exit(cmdVerify(os.Args[2:]))
	case "dashboard", "ui":
		os.Exit(cmdUI(os.Args[2:]))
	case "mcp":
		os.Exit(cmdMCP(os.Args[2:]))
	// 수집기 (collector)
	case "collector", "collect":
		os.Exit(cmdCollect(os.Args[2:]))
	case "seed":
		os.Exit(cmdSeed(os.Args[2:]))
	// deprecated aliases (pre-collect layout)
	case "sniff":
		os.Exit(cmdSniff(os.Args[2:]))
	case "convert":
		os.Exit(cmdConvert(os.Args[2:]))
	case "mirror":
		os.Exit(cmdMirror(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func cmdCollect(args []string) int {
	// 기본 = 사이드카 스니핑. 모드 하위명령은 특수 상황용(고급).
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return cmdSniff(args)
	}
	switch args[0] {
	case "agent":
		return cmdCollectAgent(args[1:])
	case "sniff":
		return cmdSniff(args[1:])
	case "pcap":
		return cmdConvert(args[1:])
	case "mirror":
		return cmdMirror(args[1:])
	case "proxy":
		return cmdCollectProxy(args[1:])
	default:
		// 모드 생략하고 바로 플래그를 준 경우
		return cmdSniff(args)
	}
}

// cmdCollectAgent runs a long-lived collector controlled by the dashboard:
// registers, heartbeats, and obeys start/stop/upload commands. Recordings are
// streamed to --dir and uploaded on demand.
func cmdCollectAgent(args []string) int {
	fs := flag.NewFlagSet("collect agent", flag.ExitOnError)
	dashboard := fs.String("dashboard", "", "dashboard URL to register with (required)")
	token := fs.String("token", "", "shared secret for the dashboard")
	name := fs.String("name", "", "collector display name (default: hostname)")
	dir := fs.String("dir", "./recordings", "local directory for recordings")
	iface := fs.String("iface", "", "default network interface for sniff (auto if empty)")
	fs.Parse(args)
	if *dashboard == "" {
		fs.Usage()
		return 2
	}
	if *iface == "" {
		if d, err := sniff.DefaultInterface(); err == nil {
			*iface = d
		}
	}
	ctx := context.Background()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		os.Exit(0)
	}()
	if err := agent.RunDaemon(ctx, *dashboard, *token, *name, *dir, nil, *iface); err != nil {
		log.Fatalf("agent: %v", err)
	}
	return 0
}

// cmdCollectProxy runs the dev recording proxy headless (no UI) — for remote
// staging hosts. With a config file it records full answer sheets (responses
// + per-request write-sets from the old stack's DB).
func cmdCollectProxy(args []string) int {
	fs := flag.NewFlagSet("collect proxy", flag.ExitOnError)
	configPath := fs.String("c", "", "config YAML (old stack = recording target + its MySQL)")
	fs.StringVar(configPath, "config", *configPath, "config YAML")
	listen := fs.String("listen", ":9090", "proxy listen address")
	upstream := fs.String("upstream", "", "recording target (default: old.base_url from config)")
	out := fs.String("out", "", "output file: .golden.jsonl (answer sheet) or .jsonl (requests only)")
	duration := fs.Duration("duration", 0, "stop after this long (default: until Ctrl-C)")
	fs.Parse(args)
	if *out == "" {
		fs.Usage()
		return 2
	}

	opts := capture.Options{Golden: golden.IsGoldenFile(*out)}
	if *configPath != "" {
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		if *upstream == "" {
			*upstream = cfg.Old.BaseURL
		}
		if opts.Golden {
			if src, err := wscapture.For("capture", &cfg.Old, 5599); err == nil {
				opts.Source = src
			}
			opts.Attribution = cfg.Attribution
			opts.NoiseColumns = cfg.Noise.Columns
			opts.TablesIgnore = cfg.Noise.TablesIgnore
		}
	}
	if *upstream == "" {
		log.Fatalf("--upstream (or -c with old.base_url) is required")
	}

	rec, err := capture.Start(*listen, *upstream, *out, opts)
	if err != nil {
		log.Fatalf("collect proxy: %v", err)
	}
	<-stopChannel(*duration)
	rec.Stop()
	st := rec.Status()
	log.Printf("recorded %d exchange(s) -> %s", st.Count, *out)
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `mockingbox — verify a backend rewrite behaves like the original.

Two things run:
  dashboard   the console + verifier (one place; open it in a browser)
  collect     a sidecar that records traffic where it flows, and sends it to the dashboard

Typical use (assumes an internal/trusted network):

  # 1) on the box that will hold the console + the new stack:
  mockingbox dashboard -c config.yaml
        → open http://<this-host>:8642, fill in the new stack + seed source, click Seed

  # 2) next to the running old service (its port is 8080 here):
  mockingbox collect --port 8080 --out rec.golden.jsonl --dashboard http://<console>:8642
        → records passively (never in the request path); uploads when done

  # 3) back in the dashboard: pick the recording, click Verify. Done.

More:
  mcp         MCP server (stdio) — let an AI assistant drive it conversationally
  verify/run  run a verification from the CLI instead of the dashboard button
  collect pcap|mirror|proxy   other capture sources (tcpdump file / AWS mirror / dev proxy)
  seed        seed the DB from the CLI instead of the dashboard button
  run 'mockingbox <cmd> -h' for flags
`)
}

func cmdVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	configPath := fs.String("c", "", "config YAML path (required; only `new` stack is used)")
	fs.StringVar(configPath, "config", *configPath, "config YAML path")
	goldenPath := fs.String("golden", "", "golden file (.golden.jsonl) (required)")
	baselinePath := fs.String("baseline", "", "self-check run report (run-*.json): matching diffs become NOISE")
	fs.Parse(args)
	if *configPath == "" || *goldenPath == "" {
		fs.Usage()
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	meta, entries, err := golden.Read(*goldenPath)
	if err != nil {
		log.Fatalf("golden: %v", err)
	}
	log.Printf("golden: %d entr(ies) | new=%s | per-request write-sets: %v",
		len(entries), cfg.New.BaseURL, meta.Serialized)

	// preflight: 검증 DB가 시딩된 적 있는지 (T0 정합의 최소 확인)
	if m := cfg.New.PrimaryMySQL(); m != nil {
		if marker, _ := seed.ReadMarker(m); marker == nil {
			log.Printf("⚠ 검증 DB에 seed 이력이 없습니다 — 골든의 T0 상태와 다르면 diff가 어긋납니다 (mockingbox seed 참고)")
		} else {
			log.Printf("seed marker: %s에 %s에서 시딩됨 (schemas=%s, golden=%s)",
				marker.SeededAt, marker.Source, marker.Schemas, marker.Golden)
		}
	}

	verifier := replay.NewVerifier(cfg)
	if err := verifier.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
	defer verifier.Stop()

	results := verifier.Run(meta, entries)
	if *baselinePath != "" {
		baseline, err := report.LoadRun(*baselinePath)
		if err != nil {
			log.Fatalf("baseline: %v", err)
		}
		noise := report.ApplyBaseline(results, baseline)
		log.Printf("baseline: %d result(s) reclassified as NOISE (capture artifacts)", noise)
	}
	report.PrintConsole(results)
	path, err := report.WriteJSON(results, cfg.Report.Dir, *goldenPath+" (verify)",
		"golden:"+*goldenPath, cfg.New.BaseURL)
	if err != nil {
		log.Fatalf("write report: %v", err)
	}
	log.Printf("JSON report: %s", path)

	for _, r := range results {
		if r.Verdict != diff.Match && r.Verdict != diff.Noise {
			return 1
		}
	}
	return 0
}

func cmdSniff(args []string) int {
	fs := flag.NewFlagSet("sniff", flag.ExitOnError)
	iface := fs.String("iface", "", "network interface (default: auto-detect primary)")
	port := fs.Int("port", 0, "service TCP port to listen for (required)")
	out := fs.String("out", "", "output file: .jsonl (requests) or .golden.jsonl (required)")
	duration := fs.Duration("duration", 0, "stop after this long (default: until Ctrl-C)")
	sample := fs.Float64("sample", 1.0, "keep this fraction of READ exchanges (writes always kept)")
	sampleWrites := fs.Bool("sample-writes-too", false, "also sample non-GET requests (breaks state replay!)")
	dashboard, token, name := agentFlags(fs)
	fs.Parse(args)
	if *port == 0 || *out == "" {
		fs.Usage()
		return 2
	}
	if *iface == "" {
		detected, err := sniff.DefaultInterface()
		if err != nil {
			log.Fatalf("interface auto-detect failed (%v) — pass --iface", err)
		}
		*iface = detected
		log.Printf("interface: %s (auto)", *iface)
	}
	output, err := sniff.NewOutput(*out, "sniff://"+*iface)
	if err != nil {
		log.Fatalf("output: %v", err)
	}
	output.SampleRate, output.SampleAllWrites = *sample, !*sampleWrites
	pipeline := sniff.NewPipeline(*port, output.Sink)
	reporter := linkAgent(*dashboard, *token, *name, "sniff", *out, output)

	stop := stopChannel(*duration)
	if err := sniff.RunLive(*iface, *port, pipeline, stop); err != nil {
		log.Fatalf("sniff: %v", err)
	}
	output.Close()
	count, skipped := output.Stats()
	log.Printf("captured %d exchange(s), sampled out %d -> %s", count, skipped, *out)
	reporter.Finish(*out)
	return 0
}

func cmdConvert(args []string) int {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	pcapPath := fs.String("pcap", "", ".pcap file from tcpdump/wireshark (required)")
	port := fs.Int("port", 0, "service TCP port (required)")
	out := fs.String("out", "", "output file: .jsonl or .golden.jsonl (required)")
	vxlan := fs.Bool("vxlan", false, "packets are VXLAN-encapsulated (mirrored capture)")
	sample := fs.Float64("sample", 1.0, "keep this fraction of READ exchanges (writes always kept)")
	sampleWrites := fs.Bool("sample-writes-too", false, "also sample non-GET requests (breaks state replay!)")
	fs.Parse(args)
	if *pcapPath == "" || *port == 0 || *out == "" {
		fs.Usage()
		return 2
	}
	output, err := sniff.NewOutput(*out, "pcap://"+*pcapPath)
	if err != nil {
		log.Fatalf("output: %v", err)
	}
	output.SampleRate, output.SampleAllWrites = *sample, !*sampleWrites
	defer output.Close()
	pipeline := sniff.NewPipeline(*port, output.Sink)
	if err := sniff.RunFile(*pcapPath, *port, *vxlan, pipeline); err != nil {
		log.Fatalf("convert: %v", err)
	}
	log.Printf("converted %d exchange(s) -> %s", output.Count, *out)
	return 0
}

func cmdMirror(args []string) int {
	fs := flag.NewFlagSet("mirror", flag.ExitOnError)
	listen := fs.String("listen", ":4789", "UDP listen address for VXLAN")
	port := fs.Int("port", 0, "inner service TCP port (required)")
	out := fs.String("out", "", "output file: .jsonl or .golden.jsonl (required)")
	duration := fs.Duration("duration", 0, "stop after this long (default: until Ctrl-C)")
	sample := fs.Float64("sample", 1.0, "keep this fraction of READ exchanges (writes always kept)")
	sampleWrites := fs.Bool("sample-writes-too", false, "also sample non-GET requests (breaks state replay!)")
	dashboard, token, name := agentFlags(fs)
	fs.Parse(args)
	if *port == 0 || *out == "" {
		fs.Usage()
		return 2
	}
	output, err := sniff.NewOutput(*out, "mirror://"+*listen)
	if err != nil {
		log.Fatalf("output: %v", err)
	}
	output.SampleRate, output.SampleAllWrites = *sample, !*sampleWrites
	pipeline := sniff.NewPipeline(*port, output.Sink)
	reporter := linkAgent(*dashboard, *token, *name, "mirror", *out, output)

	stop := stopChannel(*duration)
	if err := sniff.RunMirror(*listen, *port, pipeline, stop); err != nil {
		log.Fatalf("mirror: %v", err)
	}
	output.Close()
	count, _ := output.Stats()
	log.Printf("captured %d exchange(s) -> %s", count, *out)
	reporter.Finish(*out)
	return 0
}

// cmdSeed copies schemas+data from a source MySQL (PITR temp instance, dev DB)
// into the verification datastore (config `new.mysql`). Pure Go — the user
// provides connection info only; tables are created automatically.
func cmdSeed(args []string) int {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	configPath := fs.String("c", "", "config YAML (target = new.mysql) (required)")
	fs.StringVar(configPath, "config", *configPath, "config YAML")
	fromHost := fs.String("from", "", "source MySQL host:port — a PITR temp instance or dev DB (required)")
	fromUser := fs.String("from-user", "root", "source MySQL user")
	fromPassword := fs.String("from-password", "", "source MySQL password")
	schemasCSV := fs.String("schemas", "", "schemas to copy, comma-separated (default: all non-system)")
	excludeCSV := fs.String("exclude-tables", "", "tables to skip, comma-separated (e.g. huge blob tables)")
	goldenName := fs.String("golden", "", "golden this seed corresponds to (recorded for verify preflight)")
	fs.Parse(args)
	if *configPath == "" || *fromHost == "" {
		fs.Usage()
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	dst := cfg.New.PrimaryMySQL()
	if dst == nil {
		log.Fatalf("config new needs a mysql datastore as the seed target")
	}

	host, port := *fromHost, 3306
	if h, p, ok := strings.Cut(*fromHost, ":"); ok {
		host = h
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	src := &config.MySQL{Host: host, Port: port, User: *fromUser, Password: *fromPassword}

	opts := seed.Options{GoldenName: *goldenName, ExcludeTables: map[string]bool{}}
	if *schemasCSV != "" {
		opts.Schemas = strings.Split(*schemasCSV, ",")
	}
	for _, t := range strings.Split(*excludeCSV, ",") {
		if t = strings.TrimSpace(t); t != "" {
			opts.ExcludeTables[t] = true
		}
	}

	log.Printf("seed: %s -> %s (검증용 DB를 소스 상태로 재구성 — 대상의 해당 스키마는 재생성됩니다)",
		src.Addr(), dst.Addr())
	started := time.Now()
	stats, err := seed.Run(src, dst, opts)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Printf("seed done in %s: schemas=%v tables=%d rows=%d",
		time.Since(started).Round(time.Millisecond), stats.Schemas, stats.Tables, stats.Rows)
	return 0
}

// agentFlags registers the dashboard-link flags shared by collector modes.
func agentFlags(fs *flag.FlagSet) (dashboard, token, name *string) {
	dashboard = fs.String("dashboard", "", "dashboard URL to register with (Spring-Boot-Admin style; optional)")
	token = fs.String("token", "", "shared secret for the dashboard")
	name = fs.String("name", "", "collector display name (default: hostname)")
	return
}

// linkAgent connects to the dashboard and streams live counters from output.
func linkAgent(dashboard, token, name, mode, out string, output *sniff.Output) *agent.Reporter {
	reporter := agent.Connect(dashboard, token, name, mode, out, version)
	if reporter != nil && output != nil {
		go func() {
			for {
				time.Sleep(3 * time.Second)
				count, _ := output.Stats()
				reporter.Update(count, "")
			}
		}()
	}
	return reporter
}

const version = "0.4.0-dev"

func stopChannel(after time.Duration) <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		if after > 0 {
			select {
			case <-sig:
			case <-time.After(after):
			}
		} else {
			<-sig
		}
		close(stop)
	}()
	return stop
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("c", "", "config YAML path (required)")
	fs.StringVar(configPath, "config", *configPath, "config YAML path (required)")
	corpusPath := fs.String("corpus", "", "corpus file, .jsonl or .har (required)")
	fs.Parse(args)
	if *configPath == "" || *corpusPath == "" {
		fs.Usage()
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Old.BaseURL == "" {
		log.Fatalf("live-parallel mode needs old.base_url in config (record & verify does not — use `verify`)")
	}
	specs, err := corpus.Load(*corpusPath)
	if err != nil {
		log.Fatalf("corpus: %v", err)
	}
	log.Printf("corpus: %d request(s) | old=%s new=%s", len(specs), cfg.Old.BaseURL, cfg.New.BaseURL)

	runner := replay.NewRunner(cfg)
	if err := runner.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
	defer runner.Stop()

	results := runner.Run(specs)
	report.PrintConsole(results)

	path, err := report.WriteJSON(results, cfg.Report.Dir, *corpusPath, cfg.Old.BaseURL, cfg.New.BaseURL)
	if err != nil {
		log.Fatalf("write report: %v", err)
	}
	log.Printf("JSON report: %s   (mockingbox ui --report-dir %s)", path, cfg.Report.Dir)

	for _, r := range results {
		if r.Verdict != diff.Match {
			return 1
		}
	}
	return 0
}

// cmdMCP runs the MCP server (stdio) so an AI assistant can drive mocking-box
// conversationally against a running dashboard.
func cmdMCP(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	dashboard := fs.String("dashboard", "http://localhost:8642", "dashboard URL to drive")
	token := fs.String("token", "", "dashboard shared secret")
	fs.Parse(args)
	if err := mcp.NewServer(*dashboard, *token).Run(); err != nil {
		log.Fatalf("mcp: %v", err)
	}
	return 0
}

func cmdUI(args []string) int {
	fs := flag.NewFlagSet("dashboard", flag.ExitOnError)
	configPath := fs.String("c", "config.yaml", "config YAML path")
	fs.StringVar(configPath, "config", *configPath, "config YAML path")
	addr := fs.String("addr", ":8642", "listen address")
	token := fs.String("token", "", "shared secret for collector registration/upload")
	fs.Parse(args)

	if _, err := config.Load(*configPath); err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("mocking-box dashboard: http://localhost%s  (config: %s)", *addr, *configPath)
	if err := ui.Serve(*addr, *configPath, *token); err != nil {
		log.Fatalf("dashboard: %v", err)
	}
	return 0
}
