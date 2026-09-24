package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProbePayloads(t *testing.T) {
	for _, name := range []string{"png-direct", "raw-direct", "zlib-direct", "tempfile", "tempfile-missing"} {
		t.Run(name, func(t *testing.T) {
			seq, res, err := makeProbe(name, 123, settings{Action: "q"})
			if err != nil {
				t.Fatal(err)
			}
			if res != nil {
				defer res.remove()
			}
			control, body, ok := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b_G"), "\x1b\\"), ";")
			if !ok || !strings.Contains(control, "i=123,q=0,") {
				t.Fatalf("bad control %q", seq)
			}
			data, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "png-direct":
				img, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				r, g, b, a := img.At(0, 0).RGBA()
				if r != 65535 || g != 0 || b != 0 || a != 65535 {
					t.Fatal("wrong PNG pixel")
				}
			case "zlib-direct":
				if !strings.Contains(control, "o=z") {
					t.Fatal(control)
				}
				zr, err := zlib.NewReader(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				data, err = io.ReadAll(zr)
				if err != nil {
					t.Fatal(err)
				}
				_ = zr.Close()
				fallthrough
			case "raw-direct":
				if !bytes.Equal(data, []byte{255, 0, 0, 255}) {
					t.Fatalf("wrong raw pixel %v", data)
				}
			case "tempfile", "tempfile-missing":
				if string(data) != res.name || !strings.Contains(res.name, "tty-graphics-protocol") {
					t.Fatal("wrong path")
				}
				content, err := os.ReadFile(res.name)
				if name == "tempfile-missing" {
					if !os.IsNotExist(err) {
						t.Fatalf("negative control exists: %v", err)
					}
				} else if err != nil || !bytes.Equal(content, []byte{255, 0, 0, 255}) {
					t.Fatalf("file: %v %v", content, err)
				}
			}
		})
	}
}

func TestParserFragmentation(t *testing.T) {
	wire := "noise\x1b[?1;2c\x1b_Gi=7;OK\x1b\\\x1b_Gp=1,i=9;ENOENT:missing\x1b\\"
	for split := 0; split <= len(wire); split++ {
		var p parser
		replies := append(p.feed([]byte(wire[:split])), p.feed([]byte(wire[split:]))...)
		if len(replies) != 2 || replies[0].ID != 7 || replies[0].Payload != "OK" || replies[1].ID != 9 || replies[1].Payload != "ENOENT:missing" {
			t.Fatalf("split %d: %+v", split, replies)
		}
	}
}

func TestEnvironmentGate(t *testing.T) {
	for _, tc := range []struct {
		env  map[string]string
		want bool
	}{
		{map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "Apple_Terminal"}, false},
		{map[string]string{"TMUX": "yes", "NTCHARTS_TMUX_PASSTHROUGH": "true"}, false},
		{map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		{map[string]string{"TERM": "xterm-kitty"}, true},
		{map[string]string{"TERM_PROGRAM": "ghostty", "NTCHARTS_KITTY": "unsupported"}, false},
		{map[string]string{"NTCHARTS_KITTY": "supported"}, true},
	} {
		if got := envSignal(func(k string) string { return tc.env[k] }); got != tc.want {
			t.Errorf("%v: %v", tc.env, got)
		}
	}
}

func TestTmuxWrap(t *testing.T) {
	seq := "\x1b_Ga=q;AAAA\x1b\\"
	if got := wrap(seq, true); got != "\x1bPtmux;\x1b\x1b_Ga=q;AAAA\x1b\x1b\\\x1b\\" {
		t.Fatalf("%q", got)
	}
}

// The fake terminal never consumes local resources: the runner must clean them
// even after silence, rejected sends, or cancellation.
type fakeTTY struct {
	writes       bytes.Buffer
	input        []byte
	silent, fail bool
}

func (f *fakeTTY) Write(b []byte) (int, error) {
	if f.fail {
		return 0, fmt.Errorf("send failed")
	}
	f.writes.Write(b)
	var p parser
	for _, ev := range p.feed(b) {
		if !f.silent {
			f.input = append(f.input, []byte(fmt.Sprintf("\x1b_Gi=%d;OK\x1b\\", ev.ID))...)
		}
	}
	return len(b), nil
}
func (f *fakeTTY) Read(b []byte) (int, error) {
	if len(f.input) == 0 {
		time.Sleep(time.Millisecond)
		return 0, io.EOF
	}
	n := copy(b, f.input)
	f.input = f.input[n:]
	return n, nil
}

func TestRunnerCleanup(t *testing.T) {
	for _, mode := range []string{"ok", "timeout", "write-failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			tty := &fakeTTY{silent: mode == "timeout", fail: mode == "write-failure"}
			r := report{Settings: settings{Timeout: 2 * time.Millisecond, Count: 2, Batch: true, Action: "q"}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			err := runProbes(ctx, tty, &r)
			if (err != nil) != (mode == "write-failure" || mode == "cancel") {
				t.Fatalf("unexpected error %v", err)
			}
			files, err := os.ReadDir(os.Getenv("TMPDIR"))
			if err != nil || len(files) != 0 {
				t.Fatalf("leaked files: %v %v", files, err)
			}
			ids := map[int]bool{}
			for _, v := range r.Results {
				if ids[v.ID] {
					t.Fatal("reused ID")
				}
				ids[v.ID] = true
				if v.resource != nil || v.CleanupError != "" {
					t.Fatalf("cleanup failed: %+v", v)
				}
				if strings.HasPrefix(v.Response, "local error:") {
					continue
				}
				if mode == "ok" && (v.Response != "OK" || v.LatencyMS == nil) {
					t.Fatalf("missing reply: %+v", v)
				}
				if mode == "timeout" && (v.Response != "timeout" || v.LatencyMS != nil) {
					t.Fatalf("false success: %+v", v)
				}
			}
		})
	}
}
