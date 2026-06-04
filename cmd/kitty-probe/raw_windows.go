//go:build windows

package main

import (
	"fmt"
)

func setRawMode(fd int) (func(), error) {
	return nil, fmt.Errorf("raw mode not supported on Windows")
}
