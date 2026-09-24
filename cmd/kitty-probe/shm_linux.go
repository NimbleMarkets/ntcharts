//go:build linux

package main

import "os"

func openSHM(name string, create bool) (*os.File, error) {
	flags := os.O_RDONLY
	if create {
		flags = os.O_CREATE | os.O_EXCL | os.O_RDWR
	}
	return os.OpenFile("/dev/shm"+name, flags, 0600)
}
func unlinkSHM(name string) error { return os.Remove("/dev/shm" + name) }
