// kitty-probe measures transport support without involving Bubble Tea.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"text/tabwriter"
	"time"
)

type settings struct {
	Timeout time.Duration `json:"timeout_ns"`
	Count   int           `json:"count"`
	Batch   bool          `json:"batch"`
	Quiet   int           `json:"quiet"`
	Action  string        `json:"action"`
	Visual  bool          `json:"visual"`
	Hold    time.Duration `json:"hold_ns"`
	Force   bool          `json:"force"`
	Tmux    bool          `json:"tmux"`
}

type report struct {
	Time        time.Time         `json:"time"`
	Platform    string            `json:"platform"`
	Label       string            `json:"label"`
	Environment map[string]string `json:"environment"`
	Settings    settings          `json:"settings"`
	Gate        string            `json:"gate"`
	BytesSent   int               `json:"bytes_sent"`
	Error       string            `json:"error,omitempty"`
	Results     []result          `json:"results"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "kitty-probe:", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg settings
	var output, label string
	flag.DurationVar(&cfg.Timeout, "timeout", 500*time.Millisecond, "response collection window per batch (minimum 100ms)")
	flag.IntVar(&cfg.Count, "count", 1, "number of repetitions, each with fresh IDs and resources")
	flag.BoolVar(&cfg.Batch, "batch", true, "send all transports together; false sends sequentially")
	flag.IntVar(&cfg.Quiet, "q", 0, "response suppression: 0=all, 1=errors, 2=none")
	flag.StringVar(&cfg.Action, "action", "q", "q=query or T=transmit and display, then delete")
	flag.BoolVar(&cfg.Visual, "visual", false, "display a quadrant pattern (requires -action=T -batch=false)")
	flag.DurationVar(&cfg.Hold, "hold", 2*time.Second, "extra viewing time per visual probe")
	flag.BoolVar(&cfg.Force, "force", false, "explicitly bypass terminal environment gate")
	flag.BoolVar(&cfg.Tmux, "tmux", tmuxEnabled(os.Getenv), "wrap each APC in tmux passthrough")
	flag.StringVar(&output, "output", "", "also save a JSON report to this file")
	flag.StringVar(&label, "label", "", "terminal version and connection description for the report")
	flag.Parse()
	if cfg.Count < 1 || cfg.Count > 1000 || cfg.Timeout < 100*time.Millisecond || cfg.Timeout > time.Minute || cfg.Quiet < 0 || cfg.Quiet > 2 || (cfg.Action != "q" && cfg.Action != "T") || cfg.Hold < 0 || cfg.Hold > time.Minute {
		return fmt.Errorf("invalid flags: count 1..1000, timeout 100ms..1m, q 0..2, action q/T, hold 0..1m")
	}
	if cfg.Visual && (cfg.Action != "T" || cfg.Batch) {
		return fmt.Errorf("-visual requires -action=T -batch=false")
	}
	r := report{Time: time.Now().UTC(), Platform: runtime.GOOS + "/" + runtime.GOARCH, Label: label, Settings: cfg, Environment: map[string]string{}}
	for _, key := range []string{"TERM", "TERM_PROGRAM", "TERM_PROGRAM_VERSION", "NTCHARTS_KITTY", "NTCHARTS_TMUX_PASSTHROUGH"} {
		r.Environment[key] = os.Getenv(key)
	}
	// Record presence only: SSH and tmux values can contain hostnames/socket paths.
	for _, key := range []string{"TMUX", "SSH_CONNECTION", "KITTY_WINDOW_ID", "GHOSTTY_RESOURCES_DIR", "WEZTERM_PANE"} {
		if os.Getenv(key) != "" {
			r.Environment[key] = "present"
		}
	}
	r.Gate = "allowed"
	var measureErr error
	if !cfg.Force && !envSignal(os.Getenv) {
		r.Gate = "skipped: no positive terminal signal (zero terminal bytes sent)"
	} else {
		if cfg.Force {
			r.Gate = "forced"
		}
		measureErr = measure(&r)
	}
	if measureErr != nil {
		r.Error = measureErr.Error()
	}
	printReport(os.Stdout, r)
	if output != "" {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(output, append(data, '\n'), 0600); err != nil {
			return err
		}
	}
	return measureErr
}

func measure(r *report) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("terminal probing is unsupported on Windows")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open controlling terminal: %w", err)
	}
	defer tty.Close()
	restore, err := setRawMode(int(tty.Fd()))
	if err != nil {
		return err
	}
	defer restore()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runProbes(ctx, tty, r)
}

func printReport(w io.Writer, r report) {
	fmt.Fprintf(w, "%s — %s — %s\nGate: %s; terminal bytes sent: %d\n", r.Time.Format(time.RFC3339), r.Platform, r.Label, r.Gate, r.BytesSent)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "RUN\tPROBE\tID\tRESPONSE\tLATENCY ms\tRESOURCE BEFORE CLEANUP\tCLEANUP")
	for _, v := range r.Results {
		latency := "-"
		if v.LatencyMS != nil {
			latency = fmt.Sprintf("%.3f", *v.LatencyMS)
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%q\t%s\t%s\t%s\n", v.Run, v.Probe, v.ID, v.Response, latency, v.ResourceState, v.CleanupError)
	}
	_ = tw.Flush()
}

// Deliberately stricter than picture's current gate: tmux alone does not
// identify the outer terminal. -force is an explicit diagnostic override.
func envSignal(get func(string) string) bool {
	switch get("NTCHARTS_KITTY") {
	case "unsupported", "false", "0", "off", "no":
		return false
	case "supported", "true", "1", "on", "yes":
		return true
	}
	for _, key := range []string{"KITTY_WINDOW_ID", "KITTY_INSTALLATION_DIR", "GHOSTTY_RESOURCES_DIR", "WEZTERM_EXECUTABLE", "WEZTERM_PANE"} {
		if get(key) != "" {
			return true
		}
	}
	switch get("TERM") {
	case "xterm-kitty", "xterm-ghostty":
		return true
	}
	switch get("TERM_PROGRAM") {
	case "kitty", "ghostty", "WezTerm", "iTerm.app":
		return true
	}
	return false
}

func tmuxEnabled(get func(string) string) bool {
	switch get("NTCHARTS_TMUX_PASSTHROUGH") {
	case "true", "1", "on", "yes":
		return true
	case "false", "0", "off", "no":
		return false
	}
	return get("TMUX") != ""
}
