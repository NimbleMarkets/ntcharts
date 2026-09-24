package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type result struct {
	Run           int      `json:"run"`
	Probe         string   `json:"probe"`
	ID            int      `json:"id"`
	Response      string   `json:"response"`
	LatencyMS     *float64 `json:"latency_ms,omitempty"`
	ResourceState string   `json:"resource_before_cleanup"`
	CleanupError  string   `json:"cleanup_error,omitempty"`
	Replies       []reply  `json:"replies,omitempty"`
	resource      *resource
	sent          time.Time
	deleted       bool
}

type reply struct {
	ID        int     `json:"id"`
	Payload   string  `json:"payload"`
	LatencyMS float64 `json:"latency_ms"`
}

// feed accepts fragmented/coalesced APCs and discards unrelated input. Responses
// are unwrapped by tmux before reaching the application's terminal input.
type parser struct{ pending string }

func (p *parser) feed(data []byte) []reply {
	p.pending += string(data)
	var out []reply
	for {
		start := strings.Index(p.pending, "\x1b_G")
		if start < 0 {
			if len(p.pending) > 2 {
				p.pending = p.pending[len(p.pending)-2:]
			}
			break
		}
		p.pending = p.pending[start:]
		end := strings.Index(p.pending, "\x1b\\")
		if end < 0 {
			if len(p.pending) > 65536 {
				p.pending = ""
			}
			break
		}
		body := p.pending[3:end]
		p.pending = p.pending[end+2:]
		control, payload, ok := strings.Cut(body, ";")
		if !ok {
			continue
		}
		for _, field := range strings.Split(control, ",") {
			if value, ok := strings.CutPrefix(field, "i="); ok {
				id, err := strconv.Atoi(value)
				if err == nil && id > 0 {
					out = append(out, reply{ID: id, Payload: payload})
				}
			}
		}
	}
	return out
}

func runProbes(ctx context.Context, tty io.ReadWriter, r *report) (err error) {
	// Own all resources and a=T placements, including partial batches.
	defer func() {
		for i := range r.Results {
			v := &r.Results[i]
			finishResource(v)
			if e := deletePlacement(tty, r, v); e != nil {
				err = errors.Join(err, e)
			}
		}
	}()
	var p parser
	// Random per-invocation ID range, disjoint within the run; fits uint32.
	seed, _ := strconv.ParseUint(strings.TrimPrefix(uniqueName(), "/ntc-")[:6], 16, 32)
	nextID := int(seed)*64 + 100000
	for run := 1; run <= r.Settings.Count; run++ {
		var group []int
		var sequences []string
		// Construct every payload before timing the batch. Encoding and allocation
		// must not artificially inflate the measured terminal response latency.
		for _, name := range probeNames {
			nextID++
			seq, res, e := makeProbe(name, nextID, r.Settings)
			v := result{Run: run, Probe: name, ID: nextID, ResourceState: "n/a", resource: res, Response: "timeout"}
			if r.Settings.Quiet != 0 {
				v.Response = "silence (not evidence of support)"
			}
			if e != nil {
				v.Response = "local error: " + e.Error()
			}
			r.Results = append(r.Results, v)
			if e != nil {
				continue
			}
			group = append(group, len(r.Results)-1)
			sequences = append(sequences, seq)
		}
		if len(group) == 0 {
			continue
		}
		if r.Settings.Batch {
			if e := ctx.Err(); e != nil {
				return e
			}
			sent := time.Now()
			for _, idx := range group {
				r.Results[idx].sent = sent
			}
			seq := strings.Join(sequences, "")
			n, e := io.WriteString(tty, seq)
			r.BytesSent += n
			if e != nil {
				return e
			}
			if n != len(seq) {
				return io.ErrShortWrite
			}
			if e := collect(ctx, tty, &p, r, group); e != nil {
				return e
			}
		} else {
			for i, idx := range group {
				if e := ctx.Err(); e != nil {
					return e
				}
				if r.Settings.Visual {
					n, e := fmt.Fprintf(tty, "\r\n%s: red/green above blue/white\r\n", r.Results[idx].Probe)
					r.BytesSent += n
					if e != nil {
						return e
					}
				}
				r.Results[idx].sent = time.Now()
				n, e := io.WriteString(tty, sequences[i])
				r.BytesSent += n
				if e != nil {
					return e
				}
				if n != len(sequences[i]) {
					return io.ErrShortWrite
				}
				if e := collect(ctx, tty, &p, r, []int{idx}); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func collect(ctx context.Context, tty io.ReadWriter, p *parser, r *report, group []int) error {
	deadline := time.Now().Add(r.Settings.Timeout)
	if r.Settings.Visual {
		deadline = deadline.Add(r.Settings.Hold)
	}
	buf := make([]byte, 4096)
	// Full-window collection includes silence and delayed/duplicate replies.
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := tty.Read(buf)
		now := time.Now()
		if n > 0 {
			for _, ev := range p.feed(buf[:n]) {
				for i := range r.Results {
					v := &r.Results[i]
					if v.ID != ev.ID || v.sent.IsZero() {
						continue
					}
					ev.LatencyMS = float64(now.Sub(v.sent)) / float64(time.Millisecond)
					v.Replies = append(v.Replies, ev)
					if v.LatencyMS == nil {
						latency := ev.LatencyMS
						v.LatencyMS = &latency
						v.Response = ev.Payload
					}
				}
			}
		}
		// VMIN=0, VTIME=1 produces EOF on a read timeout in os.File.
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}
	for _, idx := range group {
		v := &r.Results[idx]
		finishResource(v)
		if err := deletePlacement(tty, r, v); err != nil {
			return err
		}
	}
	return nil
}

func deletePlacement(tty io.Writer, r *report, v *result) error {
	if r.Settings.Action != "T" || v.sent.IsZero() || v.deleted {
		return nil
	}
	seq := wrap(fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", v.ID), r.Settings.Tmux)
	n, err := io.WriteString(tty, seq)
	r.BytesSent += n
	if err == nil && n != len(seq) {
		err = io.ErrShortWrite
	}
	v.deleted = err == nil
	return err
}

func finishResource(v *result) {
	if v.resource == nil {
		return
	}
	res := v.resource
	present, err := res.exists()
	switch {
	case err != nil:
		v.ResourceState = "inspection error: " + err.Error()
	case strings.HasSuffix(v.Probe, "-missing"):
		v.ResourceState = "absent before send (negative control)"
	case present:
		v.ResourceState = "retained"
	default:
		v.ResourceState = "unlinked by terminal"
	}
	if err := res.remove(); err != nil && !os.IsNotExist(err) {
		v.CleanupError = err.Error()
	}
	v.resource = nil
}
