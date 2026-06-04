//go:build !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !linux && !android && !windows

package main

import (
	"fmt"
)

func setRawMode(fd int) (func(), error) {
	return nil, fmt.Errorf("raw mode not supported on this platform")
}
