//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows

package main

// flushInput is a no-op where no input flush is wired.
func flushInput(int) error { return nil }
