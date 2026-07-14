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

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/corpus"
	"github.com/pajamasi726/mocking-box/internal/diff"
	"github.com/pajamasi726/mocking-box/internal/replay"
	"github.com/pajamasi726/mocking-box/internal/report"
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
	case "ui":
		os.Exit(cmdUI(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `mockingbox — replay captured HTTP traffic against an old and a new backend,
then diff responses AND per-request DB write-sets (framework/ORM agnostic).

Commands:
  run   -c <config.yaml> --corpus <file.jsonl|.har>   replay and report (CI mode)
  ui    -c <config.yaml> [--addr :8642]               web console: capture, replay, reports, settings
`)
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
