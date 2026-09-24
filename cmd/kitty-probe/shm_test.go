//go:build linux || (darwin && cgo)

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSharedMemoryLifecycle(t *testing.T) {
	data := []byte{255, 0, 0, 255}
	r, err := shmResource(data)
	if err != nil {
		t.Skipf("shared memory unavailable: %v", err)
	}
	defer r.remove()
	if len(r.name) > 31 || !strings.HasPrefix(r.name, "/") || strings.Contains(r.name[1:], "/") {
		t.Fatalf("invalid name: %q", r.name)
	}
	f, err := openSHM(r.name, false)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	mem, err := unix.Mmap(int(f.Fd()), 0, len(data), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mem, data) {
		t.Errorf("got %v", mem)
	}
	if err := unix.Munmap(mem); err != nil {
		t.Fatal(err)
	}
	if err := r.remove(); err != nil {
		t.Fatal(err)
	}
	if present, err := r.exists(); err != nil || present {
		t.Fatalf("still exists: %v %v", present, err)
	}
	if err := r.remove(); !os.IsNotExist(err) {
		t.Fatalf("want ENOENT: %v", err)
	}
}
