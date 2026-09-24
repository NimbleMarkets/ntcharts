//go:build linux || (darwin && cgo)

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func shmResource(data []byte) (*resource, error) {
	name := uniqueName()
	f, err := openSHM(name, true)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*resource, error) {
		_ = f.Close()
		_ = unlinkSHM(name)
		return nil, err
	}
	if err := f.Truncate(int64(len(data))); err != nil {
		return fail(err)
	}
	mem, err := unix.Mmap(int(f.Fd()), 0, len(data), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return fail(err)
	}
	copy(mem, data)
	if err := unix.Munmap(mem); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		_ = unlinkSHM(name)
		return nil, err
	}
	return &resource{name: name, exists: func() (bool, error) {
		f, err := openSHM(name, false)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, f.Close()
	}, remove: func() error { return unlinkSHM(name) }}, nil
}
