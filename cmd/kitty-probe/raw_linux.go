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
	// Raw byte input/output, with signals retained so Ctrl-C restores the tty.
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.IEXTEN
	raw.Lflag |= unix.ISIG
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
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
