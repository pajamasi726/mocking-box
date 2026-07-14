// mockingbox: universal differential-testing box.
//
//	mockingbox run -c config.yaml --corpus corpus.jsonl
//	mockingbox ui  --report-dir ./report [--addr :8642]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pajamasi726/mocking-box/internal/capture"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/diff"
	"github.com/pajamasi726/mocking-box/internal/golden"
	"github.com/pajamasi726/mocking-box/internal/replay"
	"github.com/pajamasi726/mocking-box/internal/report"
	"github.com/pajamasi726/mocking-box/internal/sniff"
	"github.com/pajamasi726/mocking-box/internal/ui"
)

func main() {
	log.SetFlags(log.Ltime)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	// 분석기 (verifier)
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "verify":
		os.Exit(cmdVerify(os.Args[2:]))
	case "ui":
		os.Exit(cmdUI(os.Args[2:]))
	// 수집기 (collector)
	case "collect":
		os.Exit(cmdCollect(os.Args[2:]))
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
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "proxy":
		return cmdCollectProxy(args[1:])
	case "sniff":
		return cmdSniff(args[1:])
	case "pcap":
		return cmdConvert(args[1:])
	case "mirror":
		return cmdMirror(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown collect mode %q (proxy|sniff|pcap|mirror)\n", args[0])
		return 2
	}
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
			opts.Source = cfg.Old.MySQL
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
	fmt.Fprint(os.Stderr, `mockingbox — one binary, two roles.

COLLECTOR — record traffic into a portable recording file, where traffic flows:
  collect proxy  -c cfg.yaml --listen :9090 --out r.golden.jsonl   dev/staging: in-path proxy;
                                                                   the only mode that records
                                                                   per-request DB write-sets
  collect sniff  --iface eth0 --port 8080 --out r.golden.jsonl     prod: passive NIC tap
                                                                   (root or CAP_NET_RAW)
  collect pcap   --pcap dump.pcap --port 8080 --out r.golden.jsonl prod: convert a tcpdump file
                                                                   (nothing installed on prod)
  collect mirror --listen :4789 --port 8080 --out r.golden.jsonl   prod: AWS VPC Traffic
                                                                   Mirroring receiver (VXLAN)

VERIFIER — replay a recording and grade the new stack:
  verify  -c cfg.yaml --golden r.golden.jsonl [--baseline run.json]  new stack only, vs answer sheet
  run     -c cfg.yaml --corpus r.jsonl                               live-parallel: old AND new
  ui      -c cfg.yaml [--addr :8642]                                 web console (dashboards, dev proxy)

  --out *.golden.jsonl = answer sheet included (responses; +write-sets in proxy mode)
  --out *.jsonl        = requests only (for live-parallel runs)
  --sample 0.3         = keep 30% of reads; writes are always kept (state fidelity)
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
	iface := fs.String("iface", "", "network interface (required, e.g. eth0 / lo0)")
	port := fs.Int("port", 0, "service TCP port (required)")
	out := fs.String("out", "", "output file: .jsonl (requests) or .golden.jsonl (required)")
	duration := fs.Duration("duration", 0, "stop after this long (default: until Ctrl-C)")
	sample := fs.Float64("sample", 1.0, "keep this fraction of READ exchanges (writes always kept)")
	sampleWrites := fs.Bool("sample-writes-too", false, "also sample non-GET requests (breaks state replay!)")
	fs.Parse(args)
	if *iface == "" || *port == 0 || *out == "" {
		fs.Usage()
		return 2
	}
	output, err := sniff.NewOutput(*out, "sniff://"+*iface)
	if err != nil {
		log.Fatalf("output: %v", err)
	}
	output.SampleRate, output.SampleAllWrites = *sample, !*sampleWrites
	defer output.Close()
	pipeline := sniff.NewPipeline(*port, output.Sink)

	stop := stopChannel(*duration)
	if err := sniff.RunLive(*iface, *port, pipeline, stop); err != nil {
		log.Fatalf("sniff: %v", err)
	}
	log.Printf("captured %d exchange(s), sampled out %d -> %s", output.Count, output.Skipped, *out)
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
	defer output.Close()
	pipeline := sniff.NewPipeline(*port, output.Sink)

	stop := stopChannel(*duration)
	if err := sniff.RunMirror(*listen, *port, pipeline, stop); err != nil {
		log.Fatalf("mirror: %v", err)
	}
	log.Printf("captured %d exchange(s) -> %s", output.Count, *out)
	return 0
}

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

func cmdUI(args []string) int {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	configPath := fs.String("c", "config.yaml", "config YAML path")
	fs.StringVar(configPath, "config", *configPath, "config YAML path")
	addr := fs.String("addr", ":8642", "listen address")
	fs.Parse(args)

	if _, err := config.Load(*configPath); err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("mocking-box console: http://localhost%s  (config: %s)", *addr, *configPath)
	if err := ui.Serve(*addr, *configPath); err != nil {
		log.Fatalf("ui: %v", err)
	}
	return 0
}
