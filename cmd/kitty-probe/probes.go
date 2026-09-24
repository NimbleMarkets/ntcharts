package main

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"strings"
)

var probeNames = []string{"png-direct", "raw-direct", "zlib-direct", "tempfile", "shm", "shm-zlib", "tempfile-missing", "shm-missing"}

// Inspection precedes cleanup so terminal-side unlinking is observable.
type resource struct {
	name   string
	exists func() (bool, error)
	remove func() error
}

func uniqueName() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("/ntc-%x", b[:]) // 25 bytes, below macOS's 31-byte limit.
}

func fileResource(data []byte) (*resource, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("temporary-file transport unsupported on Windows")
	}
	f, err := os.CreateTemp("", "tty-graphics-protocol-ntc-*")
	if err != nil {
		return nil, err
	}
	r := &resource{name: f.Name(), exists: func() (bool, error) {
		_, err := os.Stat(f.Name())
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}, remove: func() error { return os.Remove(f.Name()) }}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		_ = r.remove()
		return nil, writeErr
	}
	if closeErr != nil {
		_ = r.remove()
		return nil, closeErr
	}
	return r, nil
}

func makeProbe(name string, id int, cfg settings) (string, *resource, error) {
	w := 1
	if cfg.Visual {
		w = 16
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, w))
	colors := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}}
	for y := 0; y < w; y++ {
		for x := 0; x < w; x++ {
			i := 0
			if w > 1 {
				i = (y/(w/2))*2 + x/(w/2)
			}
			img.SetNRGBA(x, y, colors[i])
		}
	}
	data := img.Pix
	opts := fmt.Sprintf("a=%s,i=%d,q=%d,f=32,s=%d,v=%d", cfg.Action, id, cfg.Quiet, w, w)
	if cfg.Action == "T" {
		opts += ",c=8,r=4,C=1"
	}
	if name == "png-direct" {
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			return "", nil, err
		}
		data = b.Bytes()
		opts = strings.Replace(opts, "f=32", "f=100", 1)
	}
	if name == "zlib-direct" || name == "shm-zlib" {
		var b bytes.Buffer
		zw, err := zlib.NewWriterLevel(&b, zlib.BestSpeed)
		if err != nil {
			return "", nil, err
		}
		if _, err := zw.Write(data); err != nil {
			return "", nil, err
		}
		if err := zw.Close(); err != nil {
			return "", nil, err
		}
		data = b.Bytes()
		opts += ",o=z"
	}
	var res *resource
	var err error
	switch name {
	case "png-direct", "raw-direct", "zlib-direct":
		opts += ",t=d"
	case "tempfile", "tempfile-missing":
		opts += ",t=t"
		res, err = fileResource(data)
	case "shm", "shm-zlib", "shm-missing":
		opts += fmt.Sprintf(",t=s,S=%d", len(data))
		res, err = shmResource(data)
	default:
		return "", nil, fmt.Errorf("unknown probe %q", name)
	}
	if err != nil {
		return "", nil, err
	}
	if res != nil {
		if strings.HasSuffix(name, "-missing") {
			if err := res.remove(); err != nil {
				return "", res, err
			}
		}
		data = []byte(res.name)
	}
	return wrap("\x1b_G"+opts+";"+base64.StdEncoding.EncodeToString(data)+"\x1b\\", cfg.Tmux), res, nil
}

func wrap(s string, tmux bool) string {
	if tmux {
		return "\x1bPtmux;" + strings.ReplaceAll(s, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return s
}
