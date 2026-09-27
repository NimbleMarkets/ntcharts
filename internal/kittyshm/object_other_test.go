//go:build (!js || !wasm) && !linux && (!darwin || !cgo)

package kittyshm

import (
	"errors"
	"testing"
)

func TestUnsupported(t *testing.T) {
	if Supported() {
		t.Fatal("shared memory is not available in this build")
	}
	if _, err := Create(4, func([]byte) error { t.Fatal("fill called"); return nil }); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	var obj *Object
	if err := obj.Unlink(); err != nil {
		t.Fatal(err)
	}
}
