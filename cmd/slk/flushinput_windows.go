//go:build windows

package main

import "golang.org/x/sys/windows"

// flushInput discards input the console has received but nobody has read.
func flushInput(fd int) error {
	return windows.FlushConsoleInputBuffer(windows.Handle(fd))
}
