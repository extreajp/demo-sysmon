package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/extreajp/demo-sysmon/internal/alert"
	"github.com/extreajp/demo-sysmon/internal/collector"
	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/httpapi"
	"github.com/extreajp/demo-sysmon/internal/output"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "once":
		err = runOnce(args)
	case "watch":
		err = runWatch(args)
	case "top":
		err = runTop(args)
	case "serve":
		err = runServe(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `sysmon — Linux host / cgroup metrics

  sysmon once   [--config PATH] [--format table|json]
  sysmon watch  [--config PATH] [--interval 1s] [--format jsonl] [--out FILE]
  sysmon top    [--config PATH] [--interval 1s]
  sysmon serve  [--config PATH] [--listen ADDR] [--cors-origin URL]
`)
}

func load(path string) (config.Config, *collector.Set, error) {
	if path == "" {
		path = "config.example.json"
	}
	cfg, err := config.Load(path)
	if err != nil && !os.IsNotExist(err) {
		return cfg, nil, err
	}
	if os.IsNotExist(err) {
		cfg = config.Config{Interval: "1s", Serve: config.Serve{Listen: "127.0.0.1:9101"}}
	}
	return cfg, collector.New(procfs.New("", ""), cfg), nil
}

func runOnce(args []string) error {
	fs := flag.NewFlagSet("once", flag.ExitOnError)
	cfgPath := fs.String("config", "config.example.json", "config path")
	format := fs.String("format", "table", "table|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, set, err := load(*cfgPath)
	if err != nil {
		return err
	}
	// first collect warms CPU deltas
	_, _ = set.Snapshot()
	time.Sleep(200 * time.Millisecond)
	snap, err := set.Snapshot()
	if err != nil {
		return err
	}
	if *format == "json" {
		return output.JSON(os.Stdout, snap)
	}
	return output.Table(os.Stdout, snap)
}

func runWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	cfgPath := fs.String("config", "config.example.json", "config path")
	interval := fs.String("interval", "", "sample interval")
	format := fs.String("format", "jsonl", "jsonl")
	outPath := fs.String("out", "", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, set, err := load(*cfgPath)
	if err != nil {
		return err
	}
	d := cfg.IntervalDuration()
	if *interval != "" {
		d, err = time.ParseDuration(*interval)
		if err != nil {
			return err
		}
	}
	w := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	_ = format
	tick := time.NewTicker(d)
	defer tick.Stop()
	for {
		snap, err := set.Snapshot()
		if err != nil {
			return err
		}
		if err := output.JSONL(w, snap); err != nil {
			return err
		}
		<-tick.C
	}
}

func runTop(args []string) error {
	fs := flag.NewFlagSet("top", flag.ExitOnError)
	cfgPath := fs.String("config", "config.example.json", "config path")
	interval := fs.String("interval", "", "refresh interval")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, set, err := load(*cfgPath)
	if err != nil {
		return err
	}
	d := cfg.IntervalDuration()
	if *interval != "" {
		d, err = time.ParseDuration(*interval)
		if err != nil {
			return err
		}
	}
	for {
		snap, err := set.Snapshot()
		if err != nil {
			return err
		}
		fmt.Print("\033[H\033[2J")
		_ = output.Table(os.Stdout, snap)
		time.Sleep(d)
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", "config.example.json", "config path")
	listen := fs.String("listen", "", "listen address")
	cors := fs.String("cors-origin", "", "CORS origin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, set, err := load(*cfgPath)
	if err != nil {
		return err
	}
	addr := cfg.Serve.Listen
	if *listen != "" {
		addr = *listen
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") {
		return fmt.Errorf("listen must be 127.0.0.1 (got %s)", addr)
	}
	origin := cfg.Serve.CORSOrigin
	if *cors != "" {
		origin = *cors
	}
	srv := &httpapi.Server{
		Collect:  set,
		Alerts:   alert.New(cfg.Alerts),
		CORS:     origin,
		Interval: cfg.IntervalDuration(),
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "sysmon listening on %s\n", addr)
	return http.Serve(ln, srv.Handler())
}
