//go:build linux || android

package main

import (
	"golang.org/x/sys/unix"
)

func setRawMode(fd int) (func(), error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	raw := *termios
	// Disable echo and canonical mode
	raw.Lflag &^= unix.ECHO | unix.ICANON
	// Set minimum read to 0, timeout to 1 tenth of a second (100ms)
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return nil, err
	}

	restore := func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, termios)
	}
	return restore, nil
}
