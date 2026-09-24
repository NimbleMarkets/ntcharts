//go:build !linux && (!darwin || !cgo)

package main

import "fmt"

func shmResource([]byte) (*resource, error) {
	return nil, fmt.Errorf("shared memory unsupported (requires Linux or macOS with CGO_ENABLED=1)")
}
