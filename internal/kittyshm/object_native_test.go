//go:build linux || (darwin && cgo)

package kittyshm

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Run in a separate process to exercise the same open/map/unlink ownership
// boundary as a terminal, rather than reading the producer's own mapping.
func TestNativeReaderProcess(t *testing.T) {
	name := os.Getenv("NTCHARTS_SHM_TEST_NAME")
	if name == "" {
		return
	}
	want, err := hex.DecodeString(os.Getenv("NTCHARTS_SHM_TEST_BYTES"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := openSHM(name, false)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() < int64(len(want)) || stat.Mode().Perm() != 0600 {
		t.Fatalf("bad stat: size=%d mode=%v", stat.Size(), stat.Mode())
	}
	mem, err := unix.Mmap(int(f.Fd()), 0, len(want), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Munmap(mem)
	if !bytes.Equal(mem, want) {
		t.Fatal("pixel mismatch")
	}
	if err := unlinkSHM(name); err != nil {
		t.Fatal(err)
	}
	// Unlink must not invalidate an already-open terminal mapping.
	if !bytes.Equal(mem, want) {
		t.Fatal("unlink invalidated reader mapping")
	}
}

func readInTerminalProcess(t *testing.T, obj *Object, want []byte) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestNativeReaderProcess$")
	cmd.Env = append(os.Environ(), "NTCHARTS_SHM_TEST_NAME="+obj.Name, "NTCHARTS_SHM_TEST_BYTES="+hex.EncodeToString(want))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reader: %v\n%s", err, out)
	}
}

func pendingBytes() int { pending.Lock(); defer pending.Unlock(); return pending.bytes }

func TestNativeObjectRoundTrip(t *testing.T) {
	if !Supported() {
		t.Fatal("native backend unavailable")
	}
	// Deliberately cross a page boundary and avoid a page-aligned size.
	want := make([]byte, 8193)
	for i := range want {
		want[i] = byte(i * 13)
	}
	obj, err := Create(len(want), func(dst []byte) error { copy(dst, want); return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Unlink()
	if !regexp.MustCompile(`^/ntc-[0-9a-f]{20}$`).MatchString(obj.Name) || obj.Size != len(want) {
		t.Fatalf("bad object: %s / %d", obj.Name, obj.Size)
	}
	readInTerminalProcess(t, obj, want)
	if err := obj.Unlink(); err != nil {
		t.Fatal(err)
	}
	if err := obj.Unlink(); err != nil {
		t.Fatal(err)
	}
	if !obj.Done() || pendingBytes() != 0 {
		t.Fatal("object accounting leaked")
	}
}

func TestNativeCreateFailure(t *testing.T) {
	fail := errors.New("fill failure")
	if _, err := Create(4, func([]byte) error { return fail }); !errors.Is(err, fail) {
		t.Fatal(err)
	}
	for _, size := range []int{-1, 0, maxPendingBytes + 1} {
		if _, err := Create(size, func([]byte) error { t.Fatal("invalid size called fill"); return nil }); err == nil {
			t.Fatal("invalid size accepted")
		}
	}
	if _, err := Create(4, nil); err == nil {
		t.Fatal("nil fill accepted")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected fill panic")
			}
		}()
		_, _ = Create(4, func([]byte) error { panic("fill panic") })
	}()
	if pendingBytes() != 0 {
		t.Fatal("failed creation leaked accounting")
	}
}

func waitDone(t *testing.T, obj *Object) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !obj.Done() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !obj.Done() {
		t.Fatal("retired object was not reaped")
	}
}

func TestNativeRetirementAllowsDelayedReader(t *testing.T) {
	want := []byte{10, 20, 30, 255}
	obj, err := Create(4, func(dst []byte) error { copy(dst, want); return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Unlink()
	obj.Retire()
	// Simulate a terminal reading after the producer has moved to a new frame.
	time.Sleep(20 * time.Millisecond)
	readInTerminalProcess(t, obj, want)
	waitDone(t, obj)
	if pendingBytes() != 0 {
		t.Fatal("consumed object leaked accounting")
	}
}

func TestNativeRetirementReapsAbandonedFrame(t *testing.T) {
	obj, err := Create(4, func([]byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Unlink()
	retireNative(obj, time.Millisecond, 20*time.Millisecond)
	waitDone(t, obj)
	f, err := openSHM(obj.Name, false)
	if err == nil {
		f.Close()
		t.Fatal("abandoned object remains openable")
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if pendingBytes() != 0 {
		t.Fatal("abandoned object leaked accounting")
	}
}

// Opt-in integration check: run this test binary inside a real local Kitty
// terminal. Observing terminal-side unlink proves the terminal read the object.
func TestNativeTerminalConsumesObject(t *testing.T) {
	if os.Getenv("NTCHARTS_SHM_TERMINAL_TEST") != "1" {
		t.Skip("requires a real local Kitty terminal")
	}
	obj, err := Create(4, func(dst []byte) error { copy(dst, []byte{20, 40, 60, 255}); return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Unlink()
	_, err = fmt.Fprintf(os.Stdout, "\x1b_Ga=q,t=s,f=32,s=1,v=1,S=4,i=2147483000,q=2;%s\x1b\\", base64.StdEncoding.EncodeToString([]byte(obj.Name)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f, err := openSHM(obj.Name, false)
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("terminal did not consume the shared-memory query")
}
