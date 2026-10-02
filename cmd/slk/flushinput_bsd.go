//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package main

import "golang.org/x/sys/unix"

// fread is FREAD from <sys/fcntl.h>: TIOCFLUSH with it flushes input only.
const fread = 1

// flushInput discards input the terminal has received but nobody has read.
func flushInput(fd int) error {
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, fread)
}
