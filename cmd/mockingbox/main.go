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
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "verify":
		os.Exit(cmdVerify(os.Args[2:]))
	case "ui":
		os.Exit(cmdUI(os.Args[2:]))
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

func usage() {
	fmt.Fprint(os.Stderr, `mockingbox — differential-testing box for backend rewrites:
replay captured traffic, diff responses AND per-request DB write-sets.

Verification:
  run     -c <config.yaml> --corpus <file.jsonl|.har>     live-parallel: replay against old AND new
  verify  -c <config.yaml> --golden <file.golden.jsonl>   record&verify: replay against new only
  ui      -c <config.yaml> [--addr :8642]                 web console (capture, replay, reports, settings)

Passive capture (box philosophy — never in the request path):
  sniff   --iface <if> --port <p> --out <file>            live NIC capture (root or CAP_NET_RAW)
  convert --pcap <file.pcap> --port <p> --out <file>      offline: convert a tcpdump capture
  mirror  --listen :4789 --port <p> --out <file>          AWS VPC Traffic Mirroring receiver (VXLAN)

  --out ending in .golden.jsonl records responses too (Record & Verify);
  plain .jsonl records requests only (live-parallel corpus).
`)
}

func cmdVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	configPath := fs.String("c", "", "config YAML path (required; only `new` stack is used)")
	fs.StringVar(configPath, "config", *configPath, "config YAML path")
	goldenPath := fs.String("golden", "", "golden file (.golden.jsonl) (required)")
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
	report.PrintConsole(results)
	path, err := report.WriteJSON(results, cfg.Report.Dir, *goldenPath+" (verify)",
		"golden:"+*goldenPath, cfg.New.BaseURL)
	if err != nil {
		log.Fatalf("write report: %v", err)
	}
	log.Printf("JSON report: %s", path)

	for _, r := range results {
		if r.Verdict != diff.Match {
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
